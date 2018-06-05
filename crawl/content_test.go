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

func TestContentAutoAddIndex(t *testing.T) {
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

func TestContentRedirect(t *testing.T) {
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

func TestContentFingerprint(t *testing.T) {
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

func TestContentAlias(t *testing.T) {
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

func TestContentVariant(t *testing.T) {
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

func TestContentVariantFingerprint(t *testing.T) {
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

func TestContentURLFragment(t *testing.T) {
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

func TestContentBodyChanged(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("index.html", "index")
	fs.SWriteFile("page/1/index.html", "page 0")
	fs.SWriteFile("page/2/index.html", "totally different")

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     "index",
			},
			"/page/1/": stringHandler{
				contType: htmlType,
				body:     "page 1",
			},
			"/page/2/": stringHandler{
				contType: htmlType,
				body:     "page 2",
			},
		}),
		Entries: []string{
			"/",
			"/page/1/",
			"/page/2/",
		},
		Output: fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	c.Equal(fs.SReadFile("index.html"), "index")
	c.Equal(fs.SReadFile("page/1/index.html"), "page 1")
	c.Equal(fs.SReadFile("page/2/index.html"), "page 2")
}

func TestContentContentError(t *testing.T) {
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

func TestContentInvalidMimeType(t *testing.T) {
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

func TestContentTransformBodyError(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/all.css": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, fs.Path("doesnt-exist.css"))
				}),
		}),
		Entries: []string{"/all.css"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.NotNil(err)
	c.Contains(err, "/all.css")
}

func TestContentFingerprintError(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/img.gif": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, fs.Path("doesnt-exist.gif"))
				}),
		}),
		Entries:     []string{"/img.gif"},
		Output:      fs.Path("."),
		Fingerprint: func(c *Content) bool { return true },
	}

	_, err := Crawl(cfg)
	c.Must.NotNil(err)
	c.Contains(err, "/img.gif")
}

func TestContentOutputCollision(t *testing.T) {
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

func TestContentRedirectLoop(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	called := false

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/entry.txt": stringHandler{
				contType: "text/plain",
				body:     `/r/`,
			},
			"/r/": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, "/r/", http.StatusFound)
				}),
		}),
		Entries: []string{"/entry.txt"},
		Output:  fs.Path("."),
		Transforms: map[string][]Transform{
			"text/plain": {
				func(cr *Crawler, cc *Content, b []byte) ([]byte, error) {
					called = true
					c.Panics(func() {
						cr.GetRel(cc, string(b)).FollowRedirects()
					})
					return b, nil
				},
			},
		},
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	c.True(called)
}

func TestContentTooManyRedirects(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	i := 0
	called := false

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/entry.txt": stringHandler{
				contType: "text/plain",
				body:     `/r/`,
			},
			"/out.txt": stringHandler{
				contType: "text/plain",
				body:     `/r/`,
			},
			"/r/": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if i > 50 {
						http.Redirect(w, r, "/out.txt", http.StatusFound)
						return
					}

					i++
					to := fmt.Sprintf("/r/%d", i)
					http.Redirect(w, r, to, http.StatusFound)
				}),
		}),
		Entries: []string{"/entry.txt"},
		Output:  fs.Path("."),
		Transforms: map[string][]Transform{
			"text/plain": {
				func(cr *Crawler, cc *Content, b []byte) ([]byte, error) {
					called = true
					c.Panics(func() {
						cr.GetRel(cc, string(b)).FollowRedirects()
					})
					return b, nil
				},
			},
		},
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	c.True(called)
}
