package crawl

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func (c *content) processHTML(resp *http.Response) {
	proc := processHTML{
		process: process{
			content: c,
		},
	}

	proc.run(resp)
}

type processHTML struct {
	process
}

func (proc *processHTML) run(resp *http.Response) {
	r := Minify.Reader("text/html", resp.Body)
	doc, err := goquery.NewDocumentFromReader(r)
	resp.Body.Close()

	if err != nil {
		proc.state.Errorf("[html] failed to read page %s: %v", proc, err)
		return
	}

	baseHref, _ := doc.Find("base").First().Attr("href")
	if baseHref != "" {
		proc.baseURL, err = url.Parse(baseHref)
		if err != nil {
			proc.state.Errorf("[html] invalid base URL %s: %v", baseHref, err)
		}
	}

	doc.Find("a[href]").Each(proc.anchor)
	doc.Find("script[src]").Each(proc.script)
	doc.Find("link[href]").Each(proc.link)

	html, err := doc.Html()
	if err != nil {
		proc.state.Errorf("[html] failed to generate html for %s: %v",
			proc, err)
		return
	}

	proc.isIndex = strings.HasSuffix(proc.url.Path, "/")
	proc.save(html)
}

func (proc processHTML) loadRelative(sURL string, hasAttr bool) *content {
	return proc.process.loadRelative(sURL)
}

func (proc processHTML) anchor(i int, sel *goquery.Selection) {
	c := proc.loadRelative(sel.Attr("href"))
	if c != nil {
		sel.SetAttr("href", c.url.String())
	}
}

func (proc processHTML) script(i int, sel *goquery.Selection) {
	c := proc.loadRelative(sel.Attr("src"))
	if c != nil {
		sel.SetAttr("src", c.url.String())
	}
}

func (proc processHTML) link(i int, sel *goquery.Selection) {
	c := proc.loadRelative(sel.Attr("href"))
	if c != nil {
		sel.SetAttr("href", c.url.String())
	}
}
