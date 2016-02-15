package main

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/flosch/pongo2"
	"github.com/russross/blackfriday"
	"github.com/thatguystone/acrylic/internal/afs"
	"github.com/thatguystone/cog/cfs"
)

type tmplVars struct {
	ss *siteState
	pg *page
}

type tmplDims struct {
	LG tmplDim
	MD tmplDim
	SM tmplDim
	XS tmplDim
}

type tmplDim struct {
	W, H int
}

func init() {
	pongo2.RegisterFilter("markdown", filterMarkdown)
}

func newTmplVars(ss *siteState, pg *page) *tmplVars {
	return &tmplVars{
		ss: ss,
		pg: pg,
	}
}

func (ac *tmplVars) Cfg() *config {
	return ac.ss.cfg
}

func (ac *tmplVars) Data(file string) interface{} {
	return ac.ss.data[file]
}

func (ac *tmplVars) Img(src string) *image {
	path := src
	if !filepath.IsAbs(src) {
		path = filepath.Join(ac.pg.src, "../", src)
	} else {
		path = filepath.Join(ac.ss.cfg.ContentDir, src)
	}

	img := ac.ss.imgs.get(path)

	if img == nil {
		ac.ss.errs.add(ac.pg.src,
			fmt.Errorf("image not found: %s (resolved to %s)", src, path))
		return nil
	}

	return img
}

func (ac *tmplVars) AllImgs() []string {
	var ret []string

	for _, img := range ac.ss.imgs.all {
		if img.inGallery {
			abs := "/" + afs.DropRoot("", ac.ss.cfg.ContentDir, img.src)
			ret = append(ret, abs)
		}
	}

	return ret
}

func (ac *tmplVars) Dims(lw, lh, mw, mh, sw, sh, xsw, xsh int) tmplDims {
	return tmplDims{
		LG: tmplDim{lw, lh},
		MD: tmplDim{mw, mh},
		SM: tmplDim{sw, sh},
		XS: tmplDim{xsw, xsh},
	}
}

func (ac *tmplVars) JSTags() *pongo2.Value {
	b := bytes.Buffer{}

	if !ac.ss.cfg.Debug {
		fmt.Fprintf(&b,
			`<script type="text/javascript" src="/%s"></script>`,
			ac.cacheBustAsset("all.js"))
	} else {
		for _, js := range ac.ss.cfg.JS {
			fmt.Fprintf(&b,
				`<script type="text/javascript" src="/%s"></script>`,
				filepath.Join(ac.ss.cfg.AssetsDir, js))
		}
	}

	return pongo2.AsSafeValue(b.String())
}

func (ac *tmplVars) CSSTags() *pongo2.Value {
	b := bytes.Buffer{}

	if !ac.ss.cfg.Debug {
		fmt.Fprintf(&b,
			`<link rel="stylesheet" href="/%s" />`,
			ac.cacheBustAsset("all.css"))
	} else {
		for _, css := range ac.ss.cfg.CSS {
			fmt.Fprintf(&b,
				`<link rel="stylesheet" href="/%s" />`,
				filepath.Join(ac.ss.cfg.AssetsDir, cfs.ChangeExt(css, ".css")))
		}
	}

	return pongo2.AsSafeValue(b.String())
}

func (ac *tmplVars) cacheBustAsset(path string) string {
	path = filepath.Join(ac.ss.cfg.AssetsDir, path)

	if !ac.ss.cfg.CacheBust {
		return path
	}

	return fmt.Sprintf("%s?%d", path, ac.ss.buildTime.Unix())
}

func filterMarkdown(in *pongo2.Value, param *pongo2.Value) (
	out *pongo2.Value,
	err *pongo2.Error) {

	out = pongo2.AsSafeValue(string(blackfriday.MarkdownCommon([]byte(in.String()))))
	return
}
