package crawl

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thatguystone/cog/check"
)

var (
	gifBin = []byte{
		0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
		0x00, 0x3b,
	}
)

type crawlTest struct {
	*check.C
	fs      *check.FS
	cleanup func()
}

func newTest(t *testing.T) crawlTest {
	c := check.New(t)
	fs, cleanup := c.FS()

	return crawlTest{
		C:       c,
		fs:      fs,
		cleanup: cleanup,
	}
}

func (ct crawlTest) exit() {
	ct.cleanup()
}

func (ct crawlTest) run(h http.Handler, entries ...string) {
	Run(Args{
		Handler:     h,
		EntryPoints: entries,
		Output:      ct.fs.Path("output"),
		Logf:        ct.Logf,
	})

	ct.fs.DumpTree("/output")
}

type stringHandler string

func (h stringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bytesHandler(h).ServeHTTP(w, r)
}

type bytesHandler []byte

func (h bytesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch filepath.Ext(r.URL.Path) {
	case ".css":
		w.Header().Set("Content-Type", "text/css")

	case ".js":
		w.Header().Set("Content-Type", "text/javascript")
	}

	w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))
	w.Write(h)
}

func TestBasic(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<link href="/static/all.css" rel="stylesheet">
			<script src="./static/all.js"></script>
			Index <a href="/test">Test</a>`))
	mux.Handle("/test/",
		stringHandler(`<!DOCTYPE html>
			Test <a href="/">Index</a>`))
	mux.Handle("/static/img.gif", bytesHandler(gifBin))
	mux.Handle("/static/img-redirect.gif",
		http.RedirectHandler("img.gif", http.StatusMovedPermanently))
	mux.Handle("/static/all.js",
		stringHandler(`alert("js!");`))
	mux.Handle("/static/all.css",
		stringHandler(`
			html {
				background: url(/static/img.gif);
			}

			a {
				background: url("/static/img-redirect.gif");
				color: #e5e5e5;
			}
		`))

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `<a href="/test/">`) // URL should be rewritten
	ct.Contains(index, `/static/all.js`)

	test := ct.fs.SReadFile("output/test/index.html")
	ct.Contains(test, `<a href="/">`) // URL should not be rewritten

	css := ct.fs.SReadFile("output/static/all.css")
	ct.Contains(css, `url("/static/img.gif`)
	ct.NotContains(css, `img-redirect.gif`)

	ct.fs.FileExists("output/static/img.gif")
}

func TestExternals(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<link href="http://example.com/EXTERNAL0" rel="stylesheet">
			<script src="http://example.com/EXTERNAL1"></script>
			<a href="http://example.com/EXTERNAL2">External</a>`))

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `http://example.com/EXTERNAL0`)
	ct.Contains(index, `http://example.com/EXTERNAL1`)
	ct.Contains(index, `http://example.com/EXTERNAL2`)
}

func TestCaching(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<img src="/img.gif">`))

	requested := false
	hasCached := false

	mux.HandleFunc("/img.gif",
		func(w http.ResponseWriter, r *http.Request) {
			requested = true
			hasCached = r.Header.Get("If-Modified-Since") != ""

			if !hasCached {
				ct.fs.WriteFile("output/img.gif", gifBin)
			}

			http.ServeFile(w, r, ct.fs.Path("output/img.gif"))
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})

	ct.Must.True(requested)
	ct.Must.False(hasCached)
	requested = false

	ct.NotPanics(func() {
		ct.run(mux)
	})

	ct.Must.True(requested)
	ct.Must.True(hasCached)
}

func TestCacheBusting(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<img src="/static/img.gif">
			<a href="page/">Page</a>`))
	mux.Handle("/page/", stringHandler(`<!DOCTYPE html>`))
	mux.Handle("/static/img.gif", bytesHandler(gifBin))

	ct.NotPanics(func() {
		ct.run(mux)
	})

	index := ct.fs.SReadFile("output/index.html")
	ct.Contains(index, `img.gif?v=`)
	ct.Contains(index, `href="/page/"`)
}

func TestOutputAsCache(t *testing.T) {
	ct := newTest(t)
	defer ct.exit()

	imgPath := ct.fs.Path("output/img.gif")
	lastMod := time.Now().Add(-time.Hour).Truncate(time.Second)

	mux := http.NewServeMux()
	mux.Handle("/",
		stringHandler(`<!DOCTYPE html>
			<img src="/img.gif">`))

	mux.HandleFunc("/img.gif",
		func(w http.ResponseWriter, r *http.Request) {
			// Write directly into the output dir: simulate that we're caching
			// there
			ct.fs.WriteFile("output/img.gif", gifBin)
			err := os.Chtimes(imgPath, lastMod, lastMod)
			ct.Must.Nil(err)

			http.ServeFile(w, r, imgPath)
		})

	ct.NotPanics(func() {
		ct.run(mux)
	})

	info, err := os.Stat(imgPath)
	ct.Must.Nil(err)
	newMod := info.ModTime()
	ct.True(lastMod.Equal(newMod), "%s != %s", lastMod, newMod)
}
