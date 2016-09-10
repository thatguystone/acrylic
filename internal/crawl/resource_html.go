package crawl

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

func (rsrc *resourceHTML) recheck(resp *response, f *os.File) error {
	return nil
}

func (rsrc *resourceHTML) process(resp *response, f *os.File) error {
	r := Minify.Reader("text/html", resp.Body)
	doc, err := goquery.NewDocumentFromReader(r)
	resp.Body.Close()

	if err != nil {
		rsrc.state.Errorf("[html] failed to read page %s: %v",
			rsrc.url, err)
		return nil
	}

	doc.Find("a[href]").Each(rsrc.updateAttr("href", false))
	doc.Find("img[src]").Each(rsrc.updateAttr("src", true))
	doc.Find("link[href]").Each(rsrc.updateAttr("href", true))
	doc.Find("script[src]").Each(rsrc.updateAttr("src", true))
	doc.Find("source[href]").Each(rsrc.updateAttr("href", true))

	html, err := doc.Html()
	if err != nil {
		rsrc.state.Errorf("[html] failed to generate html for %s: %v",
			f.Name(), err)
		return nil
	}

	_, err = io.WriteString(f, html)
	return err
}

func (rsrc resourceHTML) isIndex() bool {
	return strings.HasSuffix(rsrc.url.Path, "/")
}

func (rsrc resourceHTML) updateAttr(
	attr string,
	cacheBust bool) func(int, *goquery.Selection) {

	return func(i int, sel *goquery.Selection) {
		// Should always have attr: the selectors look for the attributes
		// specifically
		val, _ := sel.Attr(attr)

		c := rsrc.loadRelative(val)
		if c != nil {
			url := c.url.String()
			if cacheBust {
				url = c.bustedURL()
			}

			sel.SetAttr(attr, url)
		}
	}
}
