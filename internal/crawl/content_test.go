package crawl

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestContentRecheck(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	lastMod := time.Now()

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				http.ServeContent(w, r,
					"/", lastMod,
					strings.NewReader(`<!DOCTYPE html>
						<link href="/static/all.css" rel="stylesheet">
						<img src="/static/img-html.gif">
						`))
			},
		},
		testHandler{
			path: "/static/all.css",
			fn: func(w http.ResponseWriter, r *http.Request) {
				http.ServeContent(w, r,
					"all.css", lastMod,
					strings.NewReader(`
						html {
							background: url(/static/img-css.gif);
						}`))
			},
		},
		testHandler{
			path:  "/static/img-html.gif",
			bytes: gifBin,
		},
		testHandler{
			path:  "/static/img-css.gif",
			bytes: gifBin,
		})

	for i := 0; i < 3; i++ {
		ct.Logf("------- %d", i)

		ct.NotPanics(func() {
			ct.run(mux)
		})

		ct.fs.FileExists("output/index.html")
		ct.fs.FileExists("output/static/all.css")
		ct.fs.FileExists("output/static/img-html.gif")
		ct.fs.FileExists("output/static/img-css.gif")
	}
}

func TestContentOutputDeleted(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	lastMod := time.Now().Add(-time.Hour)

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				http.ServeContent(w, r,
					"/", lastMod,
					strings.NewReader(`<!DOCTYPE html> OK`))
			},
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})

	err := os.Remove(ct.fs.Path("output/index.html"))
	ct.Must.Nil(err)

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `OK`)
}

func TestContentOpaqueURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="mailto:a@stoney.io">a@stoney.io</a>`,
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `href="mailto:a@stoney.io"`)
}

func TestContentInvalidEntryURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str:  `<!DOCTYPE html>`,
		})

	ct.Panics(func() {
		ct.run(mux, "://drunk-url")
	})
}

func TestContentExternalEntryURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str:  `<!DOCTYPE html>`,
		})

	ct.Panics(func() {
		ct.run(mux, "http://example.com")
	})
}

func TestContentInvalidRelURL(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="://drunk-url">Test</a>`,
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContentRedirectLoop(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="redirect">Redirect</a>`,
		},
		testHandler{
			path: "/redirect",
			handler: http.RedirectHandler("/redirect",
				http.StatusMovedPermanently),
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContentInvalidRedirect(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			str: `<!DOCTYPE html>
				<a href="redirect">Redirect</a>`,
		},
		testHandler{
			path: "/redirect",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "://herp-derp")
				w.WriteHeader(http.StatusFound)
			},
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContentInvalidLastModified(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Header().Set("Last-Modified", "What time is it?!")
				w.WriteHeader(http.StatusOK)
			},
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})
}

func TestContentInvalidContentType(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "cookies and cake")
				w.WriteHeader(http.StatusOK)
			},
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}

func TestContent500(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := ct.mux(
		testHandler{
			path: "/",
			fn: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		})

	ct.Panics(func() {
		ct.run(mux)
	})
}
