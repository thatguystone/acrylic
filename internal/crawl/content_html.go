package crawl

import (
	"net/http"
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
		proc.baseURL, err = proc.url.Parse(baseHref)
		if err != nil {
			proc.state.Errorf("[html] invalid base URL %s: %v", baseHref, err)
		}
	}

	doc.Find("a[href]").Each(proc.updateAttr("href"))
	doc.Find("img[src]").Each(proc.updateAttr("src"))
	doc.Find("link[href]").Each(proc.updateAttr("href"))
	doc.Find("script[src]").Each(proc.updateAttr("src"))
	doc.Find("source[href]").Each(proc.updateAttr("href"))

	html, err := doc.Html()
	if err != nil {
		proc.state.Errorf("[html] failed to generate html for %s: %v",
			proc, err)
		return
	}

	proc.isIndex = strings.HasSuffix(proc.url.Path, "/")
	proc.save(html)
}

func (proc processHTML) updateAttr(attr string) func(int, *goquery.Selection) {
	return func(i int, sel *goquery.Selection) {
		val, ok := sel.Attr(attr)
		if !ok {
			return
		}

		c := proc.loadRelative(val)
		if c != nil {
			sel.SetAttr(attr, c.url.String())
		}
	}
}
