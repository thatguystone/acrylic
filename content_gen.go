package toner

import (
	"bytes"

	p2 "github.com/flosch/pongo2"
)

type contentGener interface {
	// Attempt to get a content generator from this guy, if it handles it.
	getGenerator(c *content, ext string) interface{}

	// Generates the page and return its file path.
	generatePage() (string, error)

	// Name of this content, as a human would know it
	humanName() string
}

type contentGenBase struct {
	c    *content
	rend renderer
}

type contentGenAssetBase struct {
	contentGenBase
	assetDir string
	ext      string
	render   bool
	min      Minifier
}

type contentGenPage struct {
	contentGenBase
}

type contentGenJS struct {
	contentGenAssetBase
}

type contentGenCSS struct {
	contentGenAssetBase
}

type contentGenPassthru struct {
	c *content
}

var (
	generators = []contentGener{
		contentGenPage{},
		contentGenJS{},
		contentGenCSS{},
		contentGenImg{},
		contentGenPassthru{},
	}

	contentPageRends = []renderer{
		renderMarkdown{},
		renderHTML{},
	}

	contentJSRends = []renderer{
		renderCoffee{},
		renderDart{},
		renderJS{},
	}

	contentCSSRends = []renderer{
		renderLess{},
		renderSass{},
		renderCSS{},
	}
)

func (contentGenBase) findRenderer(
	c *content,
	rends []renderer,
	ext string) (contentGenBase, bool) {

	for _, r := range rends {
		if r.renders(ext) {
			b := contentGenBase{
				c:    c,
				rend: r,
			}

			return b, true
		}
	}

	return contentGenBase{}, false
}

func (gp contentGenPage) getGenerator(c *content, ext string) interface{} {
	b, ok := gp.findRenderer(c, contentPageRends, ext)
	if !ok {
		return nil
	}

	return contentGenPage{b}
}

func (gp contentGenPage) generatePage() (string, error) {
	c := gp.c
	s := c.cs.s

	dstPath, alreadyClaimed, err := c.claimDest(".html")
	if alreadyClaimed || err != nil {
		return dstPath, err
	}

	b := bytes.Buffer{}

	err = c.templatize(&b)
	if err != nil {
		return "", err
	}

	rc, err := gp.rend.render(b.Bytes())
	if err != nil {
		return "", err
	}

	f, err := s.fCreate(dstPath)
	if err != nil {
		return "", err
	}

	defer f.Close()

	lo := s.findLayout(c.cpath, "_single")
	c.tplContext["Content"] = p2.AsSafeValue(string(rc))

	return dstPath, lo.execute(c.tplContext, f)
}

func (contentGenPage) humanName() string {
	return "page"
}

func (gjs contentGenJS) getGenerator(c *content, ext string) interface{} {
	b, ok := gjs.findRenderer(c, contentJSRends, ext)
	if !ok {
		return nil
	}

	cfg := c.cs.s.cfg
	return contentGenJS{contentGenAssetBase{
		contentGenBase: b,
		assetDir:       "js",
		ext:            ".js",
		render:         cfg.RenderJS || b.rend.alwaysRender(),
		min:            cfg.MinifyJS,
	}}
}

func (contentGenJS) humanName() string {
	return "js"
}

func (gcss contentGenCSS) getGenerator(c *content, ext string) interface{} {
	b, ok := gcss.findRenderer(c, contentCSSRends, ext)
	if !ok {
		return nil
	}

	cfg := c.cs.s.cfg
	return contentGenCSS{contentGenAssetBase{
		contentGenBase: b,
		assetDir:       "css",
		ext:            ".css",
		render:         cfg.RenderCSS || b.rend.alwaysRender(),
		min:            cfg.MinifyCSS,
	}}
}

func (contentGenCSS) humanName() string {
	return "css"
}

func (gab contentGenAssetBase) generatePage() (dstPath string, err error) {
	c := gab.c
	s := c.cs.s

	alreadyClaimed := false
	if gab.render {
		dstPath, alreadyClaimed, err = c.claimStaticDest(gab.assetDir, gab.ext)
	} else {
		dstPath, alreadyClaimed, err = c.claimStaticDest(gab.assetDir, "")
	}

	if alreadyClaimed || err != nil {
		return
	}

	b := bytes.Buffer{}

	if c.meta.has() {
		err = c.templatize(&b)
	} else {
		err = c.readAll(&b)
	}

	if err != nil {
		return
	}

	rc := b.Bytes()
	if gab.render {
		rc, err = gab.rend.render(rc)
	} else {
		if !fDestChanged(c.f.srcPath, dstPath) {
			return
		}
	}

	if err != nil {
		return
	}

	if gab.min != nil {
		rc, err = gab.min.Minify(dstPath, rc)
		if err != nil {
			return
		}
	}

	err = s.fWrite(dstPath, rc)

	return
}

func (contentGenPassthru) getGenerator(c *content, ext string) interface{} {
	return contentGenPassthru{
		c: c,
	}
}

func (gpt contentGenPassthru) generatePage() (string, error) {
	return "", nil
}

func (contentGenPassthru) humanName() string {
	return "binary blob"
}
