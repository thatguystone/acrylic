package acrylic

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/thatguystone/cog/check"
)

func TestCrawlResponsePath(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("/src/index.html", `<a href="/test">test</a>`)
	fs.SWriteFile("/src/test.html", `<img src="/static/img.gif"><img src="/static/dst.gif">`)
	fs.WriteFile("/src/img.gif", gifBin)
	fs.WriteFile("/dst/static/dst.gif", gifBin)

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		Path(w, "text/html", fs.Path("/src/index.html"))
	})
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		Path(w, "text/html", fs.Path("/src/test.html"))
	})
	r.GET("/static/img.gif", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		Path(w, "", fs.Path("/src/img.gif"))
	})
	r.GET("/static/dst.gif", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		Path(w, "", fs.Path("/dst/static/dst.gif"))
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/dst"),
		Logf:    c.Logf,
	}
	c.Nil(cr.Do())

	c.Contains(fs.SReadFile("/dst/index.html"), `href="/test"`)
	c.Contains(fs.SReadFile("/dst/test"), `img.gif`)
	fs.FileExists("/dst/static/img.gif")
}

func TestCrawlResponseInvalidContentType(t *testing.T) {
	c := check.New(t)

	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "::derp")
	_, err := newResponse(w)
	c.NotNil(err)
}

func TestCrawlResponseInvalidCrawlPath(t *testing.T) {
	c := check.New(t)

	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", pathContentType)
	w.Write([]byte("{"))

	_, err := newResponse(w)
	c.NotNil(err)
}

func TestCrawlResponseSaveToErrors(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("a/b/c", "c")

	cr := new(Crawl)
	tests := []struct {
		src, dst string
	}{
		{
			src: "doesnotexist",
		},
		{
			src: "a/b/c",
			dst: "a/b/c/d",
		},
		{
			src: "a/b",
			dst: "a/b/c/d",
		},
	}

	for _, test := range tests {
		resp := response{}
		resp.body.path = filepath.Join(fs.GetDataDir(), test.src)

		err := resp.saveTo(cr, filepath.Join(fs.GetDataDir(), test.dst))
		c.NotNil(err, "src=%s, dst=%s", test.src, test.dst)
	}
}
