package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/russross/blackfriday"
)

type tmplAC struct {
	s  *site
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

func newTmplAC(s *site, pg *page) *tmplAC {
	return &tmplAC{
		s:  s,
		pg: pg,
	}
}

func (ac *tmplAC) Cfg() *config {
	return ac.s.cfg
}

func (ac *tmplAC) Data(file string) interface{} {
	return ac.s.ss.data[file]
}

func (ac *tmplAC) Img(src string) *image {
	path := src
	if !strings.HasPrefix(src, ac.s.cfg.ContentDir) {
		path = filepath.Join(ac.pg.src, "../", src)
	}

	img := ac.s.ss.imgs.get(path)

	if img == nil {
		ac.s.errs.add(ac.pg.src,
			fmt.Errorf("image not found: %s (resolved to %s)", src, path))
		return nil
	}

	return img
}

func (ac *tmplAC) AllImgs() []string {
	var ret []string

	for _, img := range ac.s.ss.imgs.all {
		if img.inGallery {
			ret = append(ret, img.src)
		}
	}

	return ret
}

func (ac *tmplAC) Dims(lw, lh, mw, mh, sw, sh, xsw, xsh int) tmplDims {
	return tmplDims{
		LG: tmplDim{lw, lh},
		MD: tmplDim{mw, mh},
		SM: tmplDim{sw, sh},
		XS: tmplDim{xsw, xsh},
	}
}

func (ac *tmplAC) JSTags() *pongo2.Value {
	b := bytes.Buffer{}

	if !ac.s.cfg.Debug {
		fmt.Fprintf(&b,
			`<script type="text/javascript" src="/%s?%d"></script>`,
			filepath.Join(ac.s.cfg.AssetsDir, "all.js"), ac.s.ss.buildTime.Unix())
	} else {
		for _, js := range ac.s.cfg.JS {
			fmt.Fprintf(&b,
				`<script type="text/javascript" src="/%s"></script>`,
				filepath.Join(ac.s.cfg.AssetsDir, js))
		}
	}

	return pongo2.AsSafeValue(b.String())
}

func (ac *tmplAC) CSSTags() *pongo2.Value {
	b := bytes.Buffer{}

	if !ac.s.cfg.Debug {
		fmt.Fprintf(&b,
			`<link rel="stylesheet" href="/%s?%d" />`,
			filepath.Join(ac.s.cfg.AssetsDir, "all.css"), ac.s.ss.buildTime.Unix())
	} else {
		for _, css := range ac.s.cfg.CSS {
			fmt.Fprintf(&b,
				`<link rel="stylesheet" href="/%s" />`,
				filepath.Join(ac.s.cfg.AssetsDir, fChangeExt(css, ".css")))
		}
	}

	return pongo2.AsSafeValue(b.String())
}

func (ac *tmplAC) cacheBuster(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		ac.s.errs.add(path, fmt.Errorf("failed to gen cache buster: %v", err))
		return ""
	}

	return fmt.Sprintf("?%d", info.ModTime().Unix())
}

func filterMarkdown(in *pongo2.Value, param *pongo2.Value) (
	out *pongo2.Value,
	err *pongo2.Error) {

	out = pongo2.AsSafeValue(string(blackfriday.MarkdownCommon([]byte(in.String()))))
	return
}
