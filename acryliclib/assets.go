package acryliclib

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
	orderChecks []assetOrderCheck

	mtx       sync.Mutex
	paths     []string            // Order to concat in combined
	havePaths map[string]struct{} // Quick lookup
}

type assetOrderCheck struct {
	srcPath string
	files   []string
}

type assetOrdering struct {
	js  []string
	css []string
}

type tagWriter interface {
	writeTag(path string, w io.Writer) error
}

type asseter interface {
	renderer
	tagWriter
	contentType() contentType
	getBase() interface{}
}

const (
	combinedName = "all"
)

// TODO(astone): asset trailers for less/coffee/etc

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

func (a *assets) addAndWrite(
	astOrd *assetOrdering,
	dstPath, relPath string,
	w io.Writer) (err error) {

	var asstr asseter
	ext := filepath.Ext(dstPath)
	for _, tagWriter := range asseters {
		if tagWriter.renders(ext) {
			asstr = tagWriter
			break
		}
	}

	if asstr == nil {
		panic(fmt.Errorf("unrecognized generated asset: %s", dstPath))
	}

	writeTag := true

	switch asstr.contentType() {
	case contJS:
		writeTag = !a.js.doCombine
		a.js.addPath(&astOrd.js, dstPath)

	case contCSS:
		writeTag = !a.css.doCombine
		a.css.addPath(&astOrd.css, dstPath)
	}

	if writeTag {
		err = asstr.writeTag(relPath, w)
	}

	return
}

func (a *assets) crunch() {
	// ok := a.verifyOrdering()
	ok := true

	if ok {
		ok = a.js.combine()
	}

	if ok {
		ok = a.css.combine()
	}

	a.js.minify()
	a.css.minify()
}

// func (a *assets) verifyOrdering() bool {
// 	if !a.js.verifyOrder && !a.css.verifyOrder {
// 		return true
// 	}

// 	for _, c := range a.s.cs.srcs {
// 		ok, failedAt := a.js.verifyOrdering(c.orderedJS)
// 		if ok {
// 			ok, failedAt = a.css.verifyOrdering(c.orderedCSS)
// 		}

// 		if !ok {
// 			a.s.errs.add(c.f.srcPath,
// 				fmt.Errorf("asset ordering inconsistent with %s. "+
// 					"You probably ordered the assets differently in another file.",
// 					failedAt))
// 			return false
// 		}
// 	}

// 	return true
// }

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
	at.havePaths = map[string]struct{}{}
}

func (at *assetType) addToOrderCheck(srcPath string, ord []string) {
	if len(ord) == 0 {
		return
	}

	at.mtx.Lock()

	at.orderChecks = append(at.orderChecks, assetOrderCheck{
		srcPath: srcPath,
		files:   ord,
	})

	at.mtx.Unlock()
}

func (at *assetType) addPath(ord *[]string, path string) {
	at.mtx.Lock()

	if _, ok := at.havePaths[path]; !ok {
		at.paths = append(at.paths, path)
		at.havePaths[path] = struct{}{}
	}

	// for i, oa := range at.paths {
	// 	if i != oa.i {
	// 		panic(fmt.Errorf("i mismatch at %d: got %d", i, oa.i))
	// 	}
	// }

	if ord != nil {
		*ord = append(*ord, path)
	}

	at.mtx.Unlock()
}

// func (at *assetType) verifyOrdering(deps []string) (ok bool, failedAt string) {
// 	if !at.verifyOrder {
// 		return true, ""
// 	}

// 	prev := 0
// 	for _, dep := range deps {
// 		oa := at.havePaths[dep]
// 		if oa.i < prev {
// 			return false, oa.path
// 		}

// 		prev = oa.i
// 	}

// 	return true, ""
// }

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

	for _, path := range at.paths {
		f, err := os.Open(path)
		if err != nil {
			at.s.errs.add(dest, err)
			return
		}

		_, err = io.Copy(fa, f)
		f.Close()

		if err != nil {
			at.s.errs.add(dest, fmt.Errorf("while copying %s: %v", path, err))
			return
		}
	}

	// Clear so that minificaton gets the right file
	at.paths = nil
	at.havePaths = map[string]struct{}{}
	at.addPath(nil, dest)

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

	for _, path := range at.paths {
		err := at.min.Minify(path)
		if err != nil {
			at.s.errs.add(path, fmt.Errorf("minification failed: %v", err))
		}
	}
}
