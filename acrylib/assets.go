package acrylib

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type assets struct {
	s *site

	// Global asset listings
	js  assetType
	css assetType

	mtx          sync.Mutex
	usedAsseters map[asseter]struct{}
}

type assetType struct {
	s           *site
	c           *content
	which       string
	doCombine   bool
	doMin       bool
	min         Minifier
	verifyOrder bool
	orderChecks []assetOrderCheck

	mtx      sync.Mutex
	paths    []*orderedAsset          // Order to concat in combined
	pathsIdx map[string]*orderedAsset // Quick lookup
}

type assetOrderCheck struct {
	srcPath string
	files   []string
}

type assetOrdering struct {
	isPage bool // If this is for a page that's being generated
	js     []string
	css    []string
}

type orderedAsset struct {
	path string
	i    int
}

type tagWriter interface {
	writeTag(path string, w io.Writer) (int, error)
}

type asseter interface {
	renderer
	tagWriter
	contentType() contentType
	writeTrailer(cfg *Config, w io.Writer) (int, error)
}

const (
	combinedName = "all"
)

var (
	asseters = []asseter{
		renderCoffee{},
		renderDart{},
		renderJS{},

		renderLess{},
		renderSass{},
		renderCSS{},
	}
)

func (a *assets) init(s *site) {
	a.s = s

	singleJS := s.cfg.RenderJS && s.cfg.SingleJS
	a.js.init(s, "js",
		singleJS,
		s.cfg.RenderJS && s.cfg.MinifyJS != nil,
		s.cfg.MinifyJS,
		singleJS && !s.cfg.UnorderedJS)

	singleCSS := s.cfg.RenderCSS && s.cfg.SingleCSS
	a.css.init(s, "css",
		singleCSS,
		s.cfg.RenderCSS && s.cfg.MinifyCSS != nil,
		s.cfg.MinifyCSS,
		singleCSS && !s.cfg.UnorderedCSS)
}

func (a *assets) getType(contType contentType) *assetType {
	switch contType {
	case contJS:
		return &a.js

	case contCSS:
		return &a.css
	}

	panic(fmt.Errorf("assets with tag content type %d not managed", contType))
}

func (a *assets) addToOrderCheck(srcPath string, assetOrd assetOrdering) {
	a.js.addToOrderCheck(srcPath, assetOrd.js)
	a.css.addToOrderCheck(srcPath, assetOrd.css)
}

func (a *assets) addByPath(astOrd *assetOrdering, dstPath string) (asstr asseter) {
	ext := filepath.Ext(dstPath)
	for _, tagWriter := range asseters {
		if tagWriter.renders(ext) {
			asstr = tagWriter
			break
		}
	}

	if asstr == nil {
		panic(fmt.Errorf("unrecognized asset: %s", dstPath))
	}

	a.addByType(astOrd, asstr.contentType(), dstPath)

	return
}

func (a *assets) addByType(astOrd *assetOrdering, contType contentType, path string) {
	switch contType {
	case contJS:
		if astOrd.isPage {
			a.js.addPath(ssLast(astOrd.js), path)
		}
		astOrd.js = append(astOrd.js, path)

	case contCSS:
		if astOrd.isPage {
			a.css.addPath(ssLast(astOrd.css), path)
		}
		astOrd.css = append(astOrd.css, path)
	}
}

func (a *assets) addToOrderingAndWrite(
	astOrd *assetOrdering,
	dstPath, relPath string,
	w io.Writer) (err error) {

	asstr := a.addByPath(astOrd, dstPath)

	writeTag := true
	switch asstr.contentType() {
	case contJS:
		writeTag = !a.js.doCombine
	case contCSS:
		writeTag = !a.css.doCombine
	}

	if writeTag {
		_, err = asstr.writeTag(relPath, w)
	}

	return
}

func (a *assets) writeTrailers(assetOrd assetOrdering, w io.Writer) error {
	pathTypes := [][]string{assetOrd.js, assetOrd.css}

OUTER:
	// Yeah, this can probably be optimized....
	for _, asstr := range asseters {
		for _, paths := range pathTypes {
			for _, path := range paths {
				if asstr.renders(filepath.Ext(path)) {
					_, err := asstr.writeTrailer(a.s.cfg, w)
					if err != nil {
						return err
					}

					continue OUTER
				}
			}
		}
	}

	return nil
}

func (a *assets) crunch() {
	ok := a.verifyOrdering()

	if ok {
		ok = a.js.combine()
	}

	if ok {
		ok = a.css.combine()
	}

	a.js.minify()
	a.css.minify()
}

func (a *assets) verifyOrdering() bool {
	ok, failedAt := a.js.verifyOrdering()

	if ok {
		ok, failedAt = a.css.verifyOrdering()
	}

	if !ok {
		a.s.errs.add(failedAt,
			fmt.Errorf("asset ordering inconsistent. "+
				"you probably ordered the assets differently in another file."))
	}

	return ok
}

func (at *assetType) init(
	s *site,
	which string,
	doCombine, doMin bool,
	min Minifier,
	verifyOrder bool) {

	at.c = &content{
		f: file{
			srcPath: fmt.Sprintf("asset-%s-placeholder", which),
		},
	}

	at.s = s
	at.which = which
	at.doCombine = doCombine
	at.doMin = doMin
	at.min = min
	at.verifyOrder = verifyOrder
	at.pathsIdx = map[string]*orderedAsset{}
}

func (at *assetType) addToOrderCheck(srcPath string, ord []string) {
	if len(ord) == 0 || !at.verifyOrder {
		return
	}

	at.mtx.Lock()

	at.orderChecks = append(at.orderChecks, assetOrderCheck{
		srcPath: srcPath,
		files:   ord,
	})

	at.mtx.Unlock()
}

func (at *assetType) addPath(prevAsset, path string) {
	at.mtx.Lock()

	push := func(p string, oa *orderedAsset) {
		if oa == nil {
			oa = &orderedAsset{
				path: p,
			}
		}
		oa.i = len(at.paths)

		at.paths = append(at.paths, oa)
		at.pathsIdx[p] = oa
	}

	if oa, ok := at.pathsIdx[path]; !ok {
		push(path, nil)
	} else if prevAsset != "" {
		// prevAsset must be in the list at this point, or things are not
		// being accounted correctly
		oap, ok := at.pathsIdx[prevAsset]
		if !ok {
			panic(fmt.Errorf("asset type %s inconsistent: %v does not contain %s",
				at.which,
				at.paths,
				prevAsset))
		}

		// If the previous asset comes after the current asset, then kick the
		// current to the back to maintain proper ordering
		if oap.i > oa.i {
			decs := at.paths[oa.i+1:]
			at.paths = append(at.paths[:oa.i], decs...)

			for _, oad := range decs {
				oad.i--
			}

			push(path, oa)
		}
	}

	// for i, oa := range at.paths {
	// 	if at.pathsIdx[oa.path].i != i {
	// 		panic(fmt.Errorf("mismatch at %s: %d (in map) != %d (in slice)",
	// 			oa.path,
	// 			at.pathsIdx[oa.path].i,
	// 			i))
	// 	}
	// }

	at.mtx.Unlock()
}

func (at *assetType) verifyOrdering() (ok bool, failedAt string) {
	if !at.verifyOrder {
		return true, ""
	}

	for _, oc := range at.orderChecks {
		prev := 0

		for _, f := range oc.files {
			oa := at.pathsIdx[f]
			if oa.i < prev {
				return false, oc.srcPath
			}

			prev = oa.i
		}
	}

	return true, ""
}

func (at *assetType) combine() (ok bool) {
	if !at.doCombine {
		return true
	}

	dest := filepath.Join(at.s.cfg.Root, at.s.cfg.PublicDir, combinedName)
	dest += "." + at.which

	_, err := at.s.cs.claimDest(dest, at.c)
	if err != nil {
		at.s.errs.add(dest, err)
		return
	}

	fa, err := at.s.fCreate(dest)
	if err != nil {
		at.s.errs.add(dest, err)
		return
	}

	defer fa.Close()

	for _, oa := range at.paths {
		var rc io.ReadCloser
		var err error
		if isRemoteURL(oa.path) {
			var resp *http.Response
			resp, err = http.Get(checkURLProtocol(oa.path))
			if err == nil {
				rc = resp.Body
			}
		} else {
			rc, err = os.Open(oa.path)
		}

		if err != nil {
			at.s.errs.add(dest, err)
			return
		}

		_, err = io.Copy(fa, rc)
		rc.Close()

		if err != nil {
			at.s.errs.add(dest, fmt.Errorf("while copying %s: %v", oa.path, err))
			return
		}

		// Remove file since it's not needed anymore
		err = os.Remove(oa.path)
		if err != nil {
			at.s.errs.add(oa.path, err)
			continue
		}

		dir := filepath.Dir(oa.path)
		f, err := os.Open(dir)
		if err != nil {
			at.s.errs.add(oa.path, err)
			continue
		}

		list, err := f.Readdir(1)
		f.Close()

		// If the dir the file was in is now empty, remove the dir, too
		if err == nil && len(list) == 0 {
			err = os.RemoveAll(dir)
		}

		if err != nil {
			at.s.errs.add(oa.path, err)
		}
	}

	err = fa.Close()
	if err != nil {
		at.s.errs.add(dest, err)
		return
	}

	// Clear so that minificaton gets the right file
	at.paths = nil
	at.pathsIdx = map[string]*orderedAsset{}
	at.addPath("", dest)

	return true
}

func (at *assetType) minify() {
	if !at.doMin {
		return
	}

	for _, oa := range at.paths {
		err := at.min.Minify(oa.path)
		if err != nil {
			at.s.errs.add(oa.path, fmt.Errorf("minification failed: %v", err))
		}
	}
}

func (ao *assetOrdering) assimilate(a *assets, oo assetOrdering) {
	if !ao.isPage || oo.isPage {
		ao.js = append(ao.js, oo.js...)
		ao.css = append(ao.css, oo.css...)
		return
	}

	for _, js := range oo.js {
		a.addByType(ao, contJS, js)
	}

	for _, css := range oo.css {
		a.addByType(ao, contCSS, css)
	}
}
