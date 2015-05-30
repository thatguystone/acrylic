package toner

import (
	"io"
	"sync"

	p2 "github.com/flosch/pongo2"
)

type assets struct {
	s    *site
	mtx  sync.Mutex
	imgs []struct{}
}

type tplAssets struct {
	*assets
	rendered bool // Ignore all future appends
	js       []string
	css      []string
}

func (a *tplAssets) append(o *tplAssets) {
	a.js = append(a.js, o.js...)
	a.css = append(a.css, o.css...)
	a.setRendered()
}

func (a *tplAssets) setRendered() {
	a.rendered = true
}

func (a *tplAssets) addJS(file string) {
	a.js = append(a.js, file)
}

func (a *tplAssets) addCSS(file string) {
	a.css = append(a.css, file)
}

func (a *tplAssets) writeJSTags(relPath string, w io.Writer) error {
	if !a.rendered {
		return nil
	}

	lo := a.s.l.find(relPath, "_js")

	for _, js := range a.js {
		err := lo.tpl.ExecuteWriter(p2.Context{"src": js}, w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *tplAssets) writeCSSTags(relPath string, w io.Writer) error {
	if !a.rendered {
		return nil
	}

	lo := a.s.l.find(relPath, "_css")

	for _, css := range a.css {
		err := lo.tpl.ExecuteWriter(p2.Context{"href": css}, w)
		if err != nil {
			return err
		}
	}

	return nil
}
