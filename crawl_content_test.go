package acrylic

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/thatguystone/cog/check"
)

func TestCrawlContentBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `
			<body>INDEX
			<a href="http://example.com">test</a>
			<a href="/test">test</a>
			<a href="">nothing</a>
			</body>`)
	})
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<body>TEST</body>`)
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/"),
		Logf:    c.Logf,
	}
	c.Nil(cr.Do())

	index := fs.SReadFile("/index.html")
	c.Contains(index, `<body>INDEX`)
	c.Contains(index, `http://example.com`)
	c.Contains(fs.SReadFile("/test"), `<body>TEST`)
}

func TestCrawlContentFingerprint(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<link rel="stylesheet" href="/style.css">`)
	})
	r.GET("/style.css", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/css")
		io.WriteString(w, `.body {}`)
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/"),
		Fingerprint: []string{
			"text/css",
		},
		Logf: c.Logf,
	}
	c.Nil(cr.Do())

	ct := cr.Get("/style.css")
	c.Equal(ct.Src.String(), "/style.css")

	rct := ct.FollowRedirects()
	c.NotEqual(rct.Src.String(), "/style.css")
	c.Contains(rct.Src.String(), "/style.")

	c.Contains(fs.SReadFile("/index.html"), rct.Src.String())
}

func TestCrawlContentFingerprintRoot(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `root`)
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/"),
		Fingerprint: []string{
			"text/html",
		},
		Logf: c.Logf,
	}
	c.Nil(cr.Do())

	ct := cr.Get("/")
	c.Equal(ct.Src.String(), "/")

	rct := ct.FollowRedirects()
	c.NotEqual(rct.Src.String(), "/index.html")
	c.Contains(rct.Src.String(), "/index.")
}

func TestCrawlContentFingerprintErrors(t *testing.T) {
	c := check.New(t)

	c.Run("InvalidBodyPath", func(c *check.C) {
		cr := new(Crawl)
		ct := newContent(cr, url.URL{
			Path: "/",
		})

		resp := response{}
		resp.body.path = "/doesntexist"
		err := ct.fingerprint(resp)
		c.NotNil(err)
	})

	c.Run("AlreadyExists", func(c *check.C) {
		cr := new(Crawl)
		ct := newContent(cr, url.URL{
			Path: "/",
		})

		resp := response{}
		resp.body.buff = bytes.NewBuffer([]byte(".body {}"))

		fp, err := fingerprint(bytes.NewReader(resp.body.buff.Bytes()))
		c.Must.Nil(err)
		fpPath := addFingerprint(addIndex(ct.Src.Path), fp)

		cr.newContent(&url.URL{
			Path: fpPath,
		})

		err = ct.fingerprint(resp)
		c.NotNil(err)
	})
}

func TestCrawlContentRelURLs(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<a href="sub/page">test</a>`)
	})
	r.GET("/sub/page", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<a href="../..">root</a>`)
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/"),
		Logf:    c.Logf,
	}
	c.Nil(cr.Do())

	c.Contains(fs.SReadFile("/index.html"), `sub/page`)
	c.Contains(fs.SReadFile("/sub/page"), `../..`)
}

func TestCrawlContentRedirects(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `
			<a href="redirect/rel">rel</a>
			<a href="redirect/abs">abs</a>
			<a href="redirect/dir/">dir</a>
			<a href="redirect/multi">multi</a>`)
	})
	r.GET("/sub/page", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `sub page`)
	})
	r.GET("/redirect/rel", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Location", "../sub/page")
		w.WriteHeader(http.StatusFound)
	})
	r.GET("/redirect/abs", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Location", "/sub/page")
		w.WriteHeader(http.StatusFound)
	})
	r.GET("/redirect/dir/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Location", "../../sub/page")
		w.WriteHeader(http.StatusFound)
	})
	r.GET("/redirect/multi", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Location", "rel")
		w.WriteHeader(http.StatusFound)
	})

	cr := Crawl{
		Handler: r,
		Output:  fs.Path("/"),
		Logf:    c.Logf,
	}
	c.Nil(cr.Do())

	index := fs.SReadFile("/index.html")
	c.Contains(index, `href="sub/page">rel`)
	c.Contains(index, `href="/sub/page">abs`)
	c.Contains(index, `href="sub/page">dir`)
	c.Contains(index, `href="sub/page">multi`)
}

func TestCrawlContentErrors(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	r := httprouter.New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `
			<body>INDEX
			<a href="/test">test</a>
			<a href="bad-redirect">test</a>
			<a href="bad-path">test</a>
			<a href="/500">500</a>
			<a href="::/">test</a>
			</body>`)
	})
	r.GET("/test", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<body>TEST</body>`)
	})
	r.GET("/bad-redirect", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Location", "::/")
		w.WriteHeader(http.StatusFound)
	})
	r.GET("/bad-path", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", pathContentType)
		io.WriteString(w, "{")
	})
	r.GET("/500", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.WriteHeader(500)
	})

	cr := Crawl{
		Handler: r,
		EntryPoints: []string{
			"/",
			"::/",
		},
		Output: fs.Path("/"),
		Logf:   c.Logf,
	}
	c.NotNil(cr.Do())
}

func TestCrawlContentCoverage(t *testing.T) {
	c := check.New(t)

	cr := new(Crawl)
	ct := newContent(cr, url.URL{})

	var resp response
	resp.contType = "text/html"
	resp.body.path = "/does/not/exist"

	err := ct.processRespBody(resp)
	c.True(os.IsNotExist(errors.Cause(err)))
}
