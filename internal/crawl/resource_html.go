package crawl

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/thatguystone/cog"
)

type resourceHTML struct {
	resourceBase
}

func (rsrc *resourceHTML) pathClaims() []string {
	paths := []string{
		rsrc.url.Path,
	}

	path := rsrc.path()
	if path != paths[0] {
		paths = append(paths, path)
	}

	return paths
}

func (rsrc *resourceHTML) path() string {
	path := rsrc.url.Path
	if rsrc.isIndex() {
		path = filepath.Join(path, "index.html")
	}

	return path
}

func (rsrc resourceHTML) isIndex() bool {
	return strings.HasSuffix(rsrc.url.Path, "/")
}

func (rsrc *resourceHTML) recheck(r io.Reader) {
	rsrc.processHTML(r)
}

func (rsrc *resourceHTML) process(resp *response) io.Reader {
	r := Minify.Reader("text/html", resp.Body)
	return rsrc.processHTML(r)
}

func (rsrc *resourceHTML) processHTML(r io.Reader) io.Reader {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		rsrc.state.Errorf("[html] failed to render page %s: %v",
			rsrc.url, err)
		return nil
	}

	doc.Find("a[href]").Each(rsrc.updateAttr("href"))
	doc.Find("img[src]").Each(rsrc.updateAttr("src"))
	doc.Find("link[href]").Each(rsrc.updateAttr("href"))
	doc.Find("script[src]").Each(rsrc.updateAttr("src"))
	doc.Find("source[href]").Each(rsrc.updateAttr("href"))

	html, err := doc.Html()
	cog.Must(err, "[html] failed to generate for %s. wtf?", rsrc.url)

	return strings.NewReader(html)
}

func (rsrc resourceHTML) updateAttr(attr string) func(int, *goquery.Selection) {
	return func(i int, sel *goquery.Selection) {
		// Should always have attr: the selectors look for the attributes
		// specifically
		val, _ := sel.Attr(attr)

		c := rsrc.loadRelative(val)
		if c != nil {
			sel.SetAttr(attr, c.url.String())
		}
	}
}
