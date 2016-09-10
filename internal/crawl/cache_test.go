package crawl

import "testing"

func TestCacheUpdate(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	ct.NotPanics(func() {
		ct.run(ct.mux(
			testHandler{
				path: "/",
				str: `<!DOCTYPE html>
				<a href="page"></a>`,
			},
			testHandler{
				path: "/page",
				str:  `<!DOCTYPE html>`,
			}))
	})

	cached := ct.fs.SReadFile("output/" + cachePath)
	ct.Contains(cached, "/page")

	ct.NotPanics(func() {
		ct.run(ct.mux(
			testHandler{
				path: "/",
				str:  `<!DOCTYPE html>`,
			}))
	})

	cached = ct.fs.SReadFile("output/" + cachePath)
	ct.NotContains(cached, "/page")
}
