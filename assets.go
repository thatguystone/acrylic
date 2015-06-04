package toner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type assets struct {
	s   *site
	js  assetType
	css assetType
}

type assetType struct {
	s           *site
	c           *content
	which       string
	doCombine   bool
	doMin       bool
	min         Minifier
	verifyOrder bool

	mtx       sync.Mutex
	paths     []*orderedAsset          // Order to concat in combined
	havePaths map[string]*orderedAsset // Quick lookup
}

type orderedAsset struct {
	path string
	i    int
}

type assetOrdering struct {
	js  []string
	css []string
}

type asseter interface {
	renderer
	getBase() interface{}
	writeTag(path string, w io.Writer) error
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

	a.js.init(s, "js",
		s.cfg.RenderJS && s.cfg.SingleJS,
		s.cfg.RenderJS && s.cfg.MinifyJS != nil,
		s.cfg.MinifyJS,
		!s.cfg.UnorderedJS)

	a.css.init(s, "css",
		s.cfg.RenderCSS && s.cfg.SingleCSS,
		s.cfg.RenderCSS && s.cfg.MinifyCSS != nil,
		s.cfg.MinifyCSS,
		!s.cfg.UnorderedCSS)
}

func (a *assets) writeTag(c *content, dstPath, relPath string, w io.Writer) (err error) {
	var ar asseter
	ext := filepath.Ext(dstPath)
	for _, tar := range asseters {
		if tar.renders(ext) {
			ar = tar
			break
		}
	}

	if ar == nil {
		panic(fmt.Errorf("unrecognized generated asset: %s", dstPath))
	}

	writeTag := true

	switch ar.getBase().(type) {
	case renderScript:
		writeTag = !a.js.doCombine
		a.js.addPath(c.kickAssets, &c.orderedJS, dstPath)

	case renderStyle:
		writeTag = !a.css.doCombine
		a.css.addPath(c.kickAssets, &c.orderedCSS, dstPath)
	}

	if writeTag {
		err = ar.writeTag(relPath, w)
	}

	return
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
	if !a.js.verifyOrder && !a.css.verifyOrder {
		return true
	}

	for _, c := range a.s.cs.srcs {
		ok, failedAt := a.js.verifyOrdering(c.orderedJS)
		if ok {
			ok, failedAt = a.css.verifyOrdering(c.orderedCSS)
		}

		if !ok {
			a.s.errs.add(c.f.srcPath,
				fmt.Errorf("assert ordering inconsistent at %s. "+
					"You probably ordered the assets differently in another file.",
					failedAt))
			return false
		}
	}

	return true
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
	at.havePaths = map[string]*orderedAsset{}
}

func (at *assetType) addPath(kickAssets bool, ordering *[]string, path string) {
	at.mtx.Lock()

	appendPath := false
	if oa, ok := at.havePaths[path]; ok {
		if kickAssets {
			decs := at.paths[oa.i+1:]
			at.paths = append(at.paths[:oa.i], decs...)
			appendPath = true

			for _, d := range decs {
				d.i--
			}
		}
	} else {
		appendPath = true
	}

	if appendPath {
		oa := &orderedAsset{
			path: path,
			i:    len(at.paths),
		}
		at.paths = append(at.paths, oa)
		at.havePaths[path] = oa
	}

	// for i, oa := range at.paths {
	// 	if i != oa.i {
	// 		panic(fmt.Errorf("i mismatch at %d: got %d", i, oa.i))
	// 	}
	// }

	if ordering != nil {
		*ordering = append(*ordering, path)
	}

	at.mtx.Unlock()
}

func (at *assetType) verifyOrdering(deps []string) (ok bool, failedAt string) {
	if !at.verifyOrder {
		return true, ""
	}

	prev := 0
	for _, dep := range deps {
		oa := at.havePaths[dep]
		if oa.i < prev {
			return false, oa.path
		}

		prev = oa.i
	}

	return true, ""
}

func (at *assetType) combine() (ok bool) {
	if !at.doCombine {
		return true
	}

	staticDir := filepath.Join(at.s.cfg.Root, at.s.cfg.PublicDir, staticPubDir)
	dest := filepath.Join(staticDir, combinedName)
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
		f, err := os.Open(oa.path)
		if err != nil {
			at.s.errs.add(dest, err)
			return
		}

		_, err = io.Copy(fa, f)
		f.Close()

		if err != nil {
			at.s.errs.add(dest, fmt.Errorf("while copying %s: %v", oa.path, err))
			return
		}
	}

	at.paths = nil
	at.havePaths = map[string]*orderedAsset{}
	at.addPath(false, nil, dest)

	err = fa.Close()
	if err != nil {
		at.s.errs.add(dest, err)
		return
	}

	err = os.RemoveAll(filepath.Join(staticDir, at.which))
	if err != nil {
		at.s.errs.add(dest, err)
		return
	}

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
