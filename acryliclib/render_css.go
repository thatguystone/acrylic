package acryliclib

import (
	"fmt"
	"io"
)

type renderStyle struct{}
type renderLess struct{ renderStyle }
type renderSass struct{ renderStyle }
type renderCSS struct{ renderStyle }

func (renderStyle) alwaysRender() bool     { return false }
func (r renderStyle) getBase() interface{} { return r }

func (renderCSS) renders(ext string) bool         { return ext == ".css" }
func (renderCSS) render(b []byte) ([]byte, error) { return b, nil }
func (renderCSS) writeTag(path string, w io.Writer) error {
	_, err := fmt.Fprintf(w, `<link rel="stylesheet" type="text/css" href="%s" />`, path)
	return err
}

func (renderLess) renders(ext string) bool         { return ext == ".less" }
func (renderLess) render(b []byte) ([]byte, error) { return b, nil }
func (renderLess) writeTag(path string, w io.Writer) error {
	_, err := fmt.Fprintf(w, `<link rel="stylesheet/less" type="text/css" href="%s" />`, path)
	return err
}

func (renderSass) renders(ext string) bool         { return ext == ".scss" }
func (renderSass) alwaysRender() bool              { return true }
func (renderSass) render(b []byte) ([]byte, error) { return b, nil }
func (renderSass) writeTag(path string, w io.Writer) error {
	_, err := fmt.Fprintf(w, `<link rel="stylesheet" type="text/css" href="%s" />`, path)
	return err
}
