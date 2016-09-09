package crawl

import (
	"net/http"
	"testing"
)

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
