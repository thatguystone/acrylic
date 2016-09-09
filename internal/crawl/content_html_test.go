package crawl

import (
	"net/http"
	"testing"
)

func TestContentHTMLBasic(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<link>
			<script></script>
			<a href=""></a>`))

	ct.NotPanics(func() {
		ct.run(mux)
	})
}

func TestContentHTMLBaseHref(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<base href="test/">
			<a href="rel-link">Link</a>
			<a href="/nested/page">Nested</a>`))
	mux.Handle("/test/rel-link",
		stringHandler(`<!DOCTYPE html>`))

	mux.Handle("/nested/page/",
		stringHandler(`<!DOCTYPE html>
			<base href="nest/">
			<a href="rel">Link</a>`))
	mux.Handle("/nested/page/nest/rel",
		stringHandler(`<!DOCTYPE html>`))

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `<a href="/test/rel-link">`)

	nested := ct.fs.SReadFile("output/nested/page/index.html")
	ct.Contains(nested, `<a href="/nested/page/nest/rel">`)
}
