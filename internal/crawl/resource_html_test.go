package crawl

import (
	"net/http"
	"testing"
)

func TestResourceHTMLBasic(t *testing.T) {
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
