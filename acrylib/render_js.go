package acrylib

import (
	"fmt"
	"io"
)

type renderScript struct{}
type renderCoffee struct{ renderScript }
type renderDart struct{ renderScript }
type renderJS struct{ renderScript }

func (renderScript) contentType() contentType                     { return contJS }
func (renderScript) alwaysRender() bool                           { return false }
func (renderScript) writeTrailer(*Config, io.Writer) (int, error) { return 0, nil }

func (renderJS) renders(ext string) bool         { return ext == ".js" }
func (renderJS) render(b []byte) ([]byte, error) { return b, nil }
func (renderJS) writeTag(path string, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<script type="text/javascript" src="%s"></script>`,
		path)
}

func (renderCoffee) render(b []byte) ([]byte, error) { return b, nil }
func (renderCoffee) renders(ext string) bool {
	switch ext {
	case ".coffee", ".litcoffee":
		return true
	}

	return false
}

func (renderCoffee) writeTag(path string, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<script type="text/coffeescript" src="%s"></script>`,
		path)
}

func (renderCoffee) writeTrailer(cfg *Config, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<script type="text/javascript" src="%s"></script>`,
		cfg.CoffeeURL)
}

func (renderDart) renders(ext string) bool         { return ext == ".dart" }
func (renderDart) render(b []byte) ([]byte, error) { return b, nil }
func (renderDart) writeTag(path string, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<script type="application/dart" src="%s"></script>`,
		path)
}
