package crawl

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thatguystone/cog/check"
)

func mux(m map[string]http.Handler) http.Handler {
	mux := http.NewServeMux()
	for path, h := range m {
		mux.Handle(path, h)
	}

	return mux
}

type stringHandler struct {
	contType string
	body     string
}

func (h stringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", h.contType)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, h.body)
}

func TestCrawlAutoAddIndex(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `index`,
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	index := cont.GetPage("/")
	c.Equal(index.OutputPath, fs.Path("/index.html"))
}

func TestCrawlRedirect(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<a href="/redirect/"></a>`,
			},
			"/other-page/": stringHandler{
				contType: htmlType,
				body:     `redirected`,
			},
			"/redirect/": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, "/other-page/", http.StatusFound)
				}),
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	index := fs.SReadFile("index.html")
	c.Contains(index, "/other-page/")
}

func TestCrawlFingerprint(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<link href="all.css" rel="stylesheet">`,
			},
			"/all.css": stringHandler{
				contType: cssType,
				body:     `body { background: #000; }`,
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
		Fingerprint: func(c *Content) bool {
			return filepath.Ext(c.URL.Path) == ".css"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	allCSS := cont.GetPage("/all.css")
	c.NotLen(allCSS.Fingerprint, 0)
	c.Contains(allCSS.URL.Path, allCSS.Fingerprint)

	index := fs.SReadFile("index.html")
	c.Contains(index, allCSS.URL.Path)
}

func TestCrawlAlias(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<a href="/page/?p=1">` +
					`<a href="/page/?p=2">`,
			},
			"/page/": stringHandler{
				contType: htmlType,
				body:     `page`,
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	p1 := cont.Get("/page/?p=1")
	c.Equal(p1.URL.RawQuery, "p=1")

	p2 := cont.Get("/page/?p=2")
	c.Equal(p2.URL.RawQuery, "p=2")

	c.Equal(p1.URL.Path, p2.URL.Path)
	c.Equal(p1.OutputPath, p2.OutputPath)
	c.Equal(p1.Fingerprint, p2.Fingerprint)
}

var variantHandler = mux(map[string]http.Handler{
	"/": stringHandler{
		contType: htmlType,
		body: `` +
			`<a href="people/?who=bob">Bob</a>` +
			`<a href="people/?who=alice">Alice</a>`,
	},
	"/people/": http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.FormValue("who") {
			case "bob":
				Variant(w, "bob.html")
				stringHandler{
					contType: htmlType,
					body:     `bob is a person`,
				}.ServeHTTP(w, r)

			case "alice":
				Variant(w, "alice.html")
				stringHandler{
					contType: htmlType,
					body:     `alice is cool`,
				}.ServeHTTP(w, r)

			default:
				http.NotFound(w, r)
			}
		}),
})

func TestCrawlVariant(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: variantHandler,
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	c.Equal(cont.Get("/people/?who=bob").URL.Path, "/people/bob.html")
	c.Equal(cont.Get("/people/?who=alice").URL.Path, "/people/alice.html")

	index := fs.SReadFile("index.html")
	c.Contains(index, "people/bob.html")
	c.Contains(index, "people/alice.html")

	c.Equal(fs.SReadFile("people/bob.html"), "bob is a person")
	c.Equal(fs.SReadFile("people/alice.html"), "alice is cool")
}

func TestCrawlVariantFingerprint(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: variantHandler,
		Entries: []string{"/"},
		Output:  fs.Path("."),
		Fingerprint: func(c *Content) bool {
			return c.URL.Path != "/"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	bob := cont.Get("/people/?who=bob")
	c.NotContains(bob.URL.Path, "bob.html")
	c.NotEqual(bob.Fingerprint, "")

	alice := cont.Get("/people/?who=alice")
	c.NotContains(alice.URL.Path, "alice.html")
	c.NotEqual(alice.Fingerprint, "")

	index := fs.SReadFile("index.html")
	c.Contains(index, bob.Fingerprint)
	c.Contains(index, alice.Fingerprint)

	c.Equal(fs.SReadFile(bob.URL.Path), "bob is a person")
	c.Equal(fs.SReadFile(alice.URL.Path), "alice is cool")
}

func TestCrawlURLFragment(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	links := []string{
		"//google.com#frag-1",
		"//google.com#frag-2",
		"//othersite.com#frag",
		"/page/#frag",
		"/about.html#frag",
	}

	body := ""
	for _, link := range links {
		body += fmt.Sprintf(`<a href="%s"></a>`, link)
	}

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     body,
			},
			"/page/": stringHandler{
				contType: htmlType,
				body:     `<div id="frag"></div>`,
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	index := fs.SReadFile("index.html")
	for _, link := range links {
		c.Contains(index, link)
	}
}

func TestCrawlContentError(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: http.HandlerFunc(http.NotFound),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.NotNil(err)
	c.Equal(err.(Error)["/"][0].(ResponseError).Code, http.StatusNotFound)
}

func TestCrawlInvalidMimeType(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: "invalid; ======",
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.NotNil(err)
	c.Equal(err, Error{
		"/": {
			mime.ErrInvalidMediaParameter,
		},
	})
}

func TestCrawlOutputCollision(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	gifPrint, err := fingerprint(bytes.NewReader(gifBin))
	c.Must.Nil(err)

	gifPath := "/img." + gifPrint + ".gif"

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<img src="` + gifPath + `">` +
					`<img src="img.gif">`,
			},
			"/img.gif": stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
			gifPath: stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
		Fingerprint: func(c *Content) bool {
			return strings.Contains(c.URL.Path, "img.gif")
		},
	}

	_, err = Crawl(cfg)
	c.Must.NotNil(err)

	c.Len(err, 1)
	for _, errs := range err.(Error) {
		c.Equal(errs[0].(AlreadyClaimedError).Path, fs.Path(gifPath))
	}
}
