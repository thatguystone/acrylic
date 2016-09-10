package crawl

import "testing"

func TestStateInternalQueryStrings(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="/page/?mucho=fun">Page</a>
				<a href="/page/?mucho=boring">Different Page?</a>`,
		},
		testHandler{
			path: "/page",
			str:  `<!DOCTYPE html>`,
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `href="/page/`)
}

func TestStateClaimConflict(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="ambiguous/"></a>
				<a href="ambiguous"></a>`,
		},
		testHandler{
			path: "/ambiguous/",
			str:  `<!DOCTYPE html>`,
		},
		testHandler{
			path: "/ambiguous",
			str:  `<!DOCTYPE html>`,
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestStateDeleteUnused(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	ct.fs.WriteFile("output/img.gif", gifBin)

	mux := ct.mux(
		testHandler{
			path: "/",
			str:  `<!DOCTYPE html>`,
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})

	ct.fs.NotFileExists("output/img.gif")
}

func TestStateCorruptCache(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	ct.fs.WriteFile("output/"+cachePath, gifBin)

	mux := ct.mux(
		testHandler{
			path: "/",
			str:  `<!DOCTYPE html>`,
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}
