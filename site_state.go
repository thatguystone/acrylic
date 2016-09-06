package main

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/flosch/pongo2"
	"github.com/tdewolff/minify"
	"github.com/thatguystone/acrylic/internal/pool"
)

type siteState struct {
	*site

	min       *minify.M
	tmplSet   *pongo2.TemplateSet
	buildTime time.Time
	ok        bool

	pool *pool.Runner // Templates can require extra work, so this must be global

	mtx    sync.Mutex
	data   map[string]interface{}
	pages  pages
	imgs   images
	blobs  []*blob
	js     []string
	css    []string
	unused *unused
}

func newSiteState(s *site) *siteState {
	tmplDir := filepath.Join(s.baseDir, s.cfg.TemplatesDir)

	ss := &siteState{
		site: s,
		tmplSet: pongo2.NewSet(
			"acrylic",
			pongo2.MustNewLocalFileSystemLoader(tmplDir)),
		buildTime: time.Now(),
		ok:        true,
		data:      map[string]interface{}{},
		pages: pages{
			byCat: map[string][]*page{},
		},
		imgs: images{
			imgs:  map[string]*image{},
			byCat: map[string][]*image{},
		},
		unused: newUnused(),
	}

	ss.tmplSet.Globals.Update(pongo2.Context{
		"cfg":  s.cfg,
		"data": ss.data,
	})

	return ss
}

func (ss *siteState) build() (ok bool) {
	ss.runPool(func() {
		ss.walk(ss.cfg.DataDir, ss.loadData)
		ss.walk(ss.cfg.ContentDir, ss.loadContent)
		ss.walk(ss.cfg.AssetsDir, ss.loadAssetImages)
		ss.walk(ss.cfg.PublicDir, ss.loadPublic)
	})

	if !ss.ok {
		return
	}

	ss.loadFinished()

	ss.runPool(func() {
		ss.renderPages()
		ss.renderAssets()
		ss.copyBlobs()
	})

	if !ss.ok {
		return
	}

	ss.runPool(func() {
		ss.renderListPages()
	})

	if !ss.ok {
		return
	}

	ss.removeUnused()

	return true
}

func (ss *siteState) runPool(cb func()) {
	pool.Pool(func(r *pool.Runner) {
		ss.pool = r
		cb()
	})

	ss.pool = nil
}

func (ss *siteState) errorf(format string, args ...interface{}) {
	ss.ok = false
	ss.log.Errorf(format, args...)
}

func (ss *siteState) markUsed(dst string) {
	ss.mtx.Lock()
	ss.unused.used(dst)
	ss.mtx.Unlock()
}

func (ss *siteState) removeUnused() {
	ss.unused.remove()
}

func (ss *siteState) loadFinished() {
	ss.pages.sort()
	ss.imgs.sort()
}

func (ss *siteState) addPage(p *page) {
	ss.mtx.Lock()
	ss.pages.add(p)
	ss.mtx.Unlock()
}

func (ss *siteState) addImage(img *image) {
	ss.mtx.Lock()
	ss.imgs.add(img)
	ss.mtx.Unlock()
}

func (ss *siteState) addBlob(src, dst string) {
	ss.mtx.Lock()
	ss.blobs = append(ss.blobs, &blob{
		src: src,
		dst: dst,
	})
	ss.mtx.Unlock()
}

func (ss *siteState) addJS(file string) {
	ss.mtx.Lock()
	ss.js = append(ss.js, file)
	ss.mtx.Unlock()
}

func (ss *siteState) addCSS(file string) {
	ss.mtx.Lock()
	ss.css = append(ss.css, file)
	ss.mtx.Unlock()
}

func (ss *siteState) addPublic(file string) {
	ss.mtx.Lock()
	ss.unused.add(file)
	ss.mtx.Unlock()
}
