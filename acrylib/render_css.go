package acrylib

import (
	"fmt"
	"io"
)

type renderStyle struct{}
type renderLess struct{ renderStyle }
type renderSass struct{ renderStyle }
type renderCSS struct{ renderStyle }

func (renderStyle) contentType() contentType                     { return contCSS }
func (renderStyle) alwaysRender() bool                           { return false }
func (renderStyle) writeTrailer(*Config, io.Writer) (int, error) { return 0, nil }

func (renderCSS) renders(ext string) bool         { return ext == ".css" }
func (renderCSS) render(b []byte) ([]byte, error) { return b, nil }
func (renderCSS) writeTag(path string, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<link rel="stylesheet" type="text/css" href="%s" />`,
		path)
}

func (renderLess) renders(ext string) bool         { return ext == ".less" }
func (renderLess) render(b []byte) ([]byte, error) { return b, nil }
func (renderLess) writeTag(path string, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<link rel="stylesheet/less" type="text/css" href="%s" />`,
		path)
}

func (renderLess) writeTrailer(cfg *Config, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<script type="text/javascript" src="%s"></script>`,
		cfg.LessURL)
}

func (renderSass) renders(ext string) bool         { return ext == ".scss" }
func (renderSass) alwaysRender() bool              { return true }
func (renderSass) render(b []byte) ([]byte, error) { return b, nil }
func (renderSass) writeTag(path string, w io.Writer) (int, error) {
	return fmt.Fprintf(w,
		`<link rel="stylesheet" type="text/css" href="%s" />`,
		path)
}
