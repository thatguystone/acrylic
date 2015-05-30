package toner

import (
	"fmt"
	"io"
	"sync"

	p2 "github.com/flosch/pongo2"
)

type assets struct {
	s    *site
	mtx  sync.Mutex
	imgs []imgAsset
}

type tplAssets struct {
	*assets
	rendered bool // Ignore all future appends
	js       []string
	css      []string
}

type imgAsset struct {
	src  string
	w    uint
	h    uint
	crop imgCrop
}

type imgCrop int

const (
	cropLeft imgCrop = iota
	cropCentered
	cropLen
)

func (c imgCrop) String() string {
	switch c {
	case cropLeft:
		return "left"

	case cropCentered:
		return "center"
	}

	panic(fmt.Errorf("unrecognized crop value: %d", c))
}

func (tpla *tplAssets) append(o *tplAssets) {
	tpla.js = append(tpla.js, o.js...)
	tpla.css = append(tpla.css, o.css...)
	tpla.setRendered()
}

func (tpla *tplAssets) setRendered() {
	tpla.rendered = true
}

func (tpla *tplAssets) addJS(file string) {
	tpla.js = append(tpla.js, file)
}

func (tpla *tplAssets) addCSS(file string) {
	tpla.css = append(tpla.css, file)
}

func (tpla *tplAssets) addAndWriteImg(
	img imgAsset,
	relPath string,
	w io.Writer) error {

	tpla.assets.mtx.Lock()
	tpla.imgs = append(tpla.imgs, img)
	tpla.assets.mtx.Unlock()

	ctx := p2.Context{
		"src":  img.src,
		"w":    img.w,
		"h":    img.h,
		"crop": img.crop,
	}

	lo := tpla.s.l.find(relPath, "_img")
	return lo.tpl.ExecuteWriter(ctx, w)
}

func (tpla *tplAssets) writeJSTags(relPath string, w io.Writer) error {
	if !tpla.rendered {
		return nil
	}

	lo := tpla.s.l.find(relPath, "_js")

	for _, js := range tpla.js {
		err := lo.tpl.ExecuteWriter(p2.Context{"src": js}, w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tpla *tplAssets) writeCSSTags(relPath string, w io.Writer) error {
	if !tpla.rendered {
		return nil
	}

	lo := tpla.s.l.find(relPath, "_css")

	for _, css := range tpla.css {
		err := lo.tpl.ExecuteWriter(p2.Context{"href": css}, w)
		if err != nil {
			return err
		}
	}

	return nil
}
