package acryliclib

import (
	"fmt"
	"io"
)

type renderScript struct{}
type renderCoffee struct{ renderScript }
type renderDart struct{ renderScript }
type renderJS struct{ renderScript }

func (renderScript) contentType() contentType { return contJS }
func (renderScript) alwaysRender() bool       { return false }
func (r renderScript) getBase() interface{}   { return r }

func (renderJS) renders(ext string) bool         { return ext == ".js" }
func (renderJS) render(b []byte) ([]byte, error) { return b, nil }
func (renderJS) writeTag(path string, w io.Writer) error {
	_, err := fmt.Fprintf(w, `<script type="text/javascript" src="%s"></script>`, path)
	return err
}

func (renderCoffee) render(b []byte) ([]byte, error) { return b, nil }
func (renderCoffee) renders(ext string) bool {
	switch ext {
	case ".coffee", ".litcoffee":
		return true
	}

	return false
}

func (renderCoffee) writeTag(path string, w io.Writer) error {
	_, err := fmt.Fprintf(w, `<script type="text/coffeescript" src="%s"></script>`, path)
	return err
}

func (renderDart) renders(ext string) bool         { return ext == ".dart" }
func (renderDart) render(b []byte) ([]byte, error) { return b, nil }
func (renderDart) writeTag(path string, w io.Writer) error {
	_, err := fmt.Fprintf(w, `<script type="application/dart" src="%s"></script>`, path)
	return err
}
