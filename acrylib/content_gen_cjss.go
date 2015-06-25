package acrylib

import (
	"bytes"
	"path/filepath"
)

type contentGenCJSS struct {
	what     string
	ext      string
	rend     renderer
	doRender bool
	statsAdd func()
}

var (
	contentJSRends = []renderer{
		renderCoffee{},
		renderJS{},
	}

	contentCSSRends = []renderer{
		renderLess{},
		renderSass{},
		renderCSS{},
	}
)

func getContentJSGener(s *site, c *content, ext string) (contentGener, contentType) {
	return getContentCJSSGener(
		"js", contJS,
		contentJSRends,
		s.cfg.RenderJS,
		s.stats.addJS,
		s, c, ext)
}

func getContentCSSGener(s *site, c *content, ext string) (contentGener, contentType) {
	return getContentCJSSGener(
		"css", contCSS,
		contentCSSRends,
		s.cfg.RenderCSS,
		s.stats.addCSS,
		s, c, ext)
}

func getContentCJSSGener(
	what string,
	contType contentType,
	renderers []renderer,
	render bool,
	statsAdd func(),
	s *site, c *content, ext string) (contentGener, contentType) {

	rend := findRenderer(ext, renderers)
	if rend == nil {
		return nil, contInvalid
	}

	cjss := &contentGenCJSS{
		what:     what,
		ext:      "." + what,
		rend:     rend,
		doRender: render || rend.alwaysRender(),
		statsAdd: statsAdd,
	}

	return cjss, contType
}

func (cjss *contentGenCJSS) defaultGen() bool {
	return false
}

func (cjss *contentGenCJSS) finalExt(c *content) string {
	if cjss.doRender {
		return cjss.ext
	}

	return filepath.Ext(c.f.srcPath)
}

func (cjss *contentGenCJSS) render(s *site, c *content) (content []byte, err error) {
	b := bytes.Buffer{}

	if c.meta.has() {
		err = c.templatize(&b)
	} else {
		err = c.readAll(&b)
	}

	if err != nil {
		return
	}

	content = b.Bytes()
	if cjss.doRender {
		content, err = cjss.rend.render(content)
	}

	return
}

func (cjss *contentGenCJSS) generate(
	content []byte,
	dstPath string,
	s *site,
	c *content) (wroteOwnFile bool, err error) {

	cjss.statsAdd()

	// If it's just a static file, don't bother copying if its src is unchanged
	dynamicFile := !c.meta.has() && !cjss.doRender
	if !dynamicFile {
		wroteOwnFile = !fSrcChanged(c.f.srcPath, dstPath)
	}

	return
}
