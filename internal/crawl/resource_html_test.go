package crawl

import "testing"

func TestResourceHTMLBasic(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<link>
				<script></script>
				<a href=""></a>`,
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})
}
