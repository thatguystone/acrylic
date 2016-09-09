package crawl

import (
	"net/http"
	"testing"
)

func TestCrawlStateInternalQueryStrings(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<a href="/page/?mucho=fun">Page</a>
			<a href="/page/?mucho=boring">Different Page?</a>`))
	mux.Handle("/page",
		stringHandler(`<!DOCTYPE html>`))

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `href="/page/`)
}

func TestCrawlStateClaimConflict(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<a href="ambiguous/"></a>
			<a href="ambiguous"></a>`))
	mux.Handle("/ambiguous/", stringHandler(`<!DOCTYPE html>`))
	mux.Handle("/ambiguous", stringHandler(`<!DOCTYPE html>`))

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestCrawlStateDeleteUnused(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	ct.fs.WriteFile("output/img.gif", gifBin)

	mux := http.NewServeMux()
	mux.Handle("/", stringHandler(`<!DOCTYPE html>`))

	ct.NotPanics(func() {
		ct.run(mux)
	})

	ct.fs.NotFileExists("output/img.gif")
}
