package crawl

import (
	"bytes"
	"fmt"
	"io"
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

	ns := newTestNS(c, nil)
	defer ns.clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `index`,
			},
		}),
		Entries: []string{"/"},
		Output:  ns.path("/public"),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	index := cont.GetPage("/")
	c.Equal(index.OutputPath, ns.path("/public/index.html"))
}

func TestContentRedirect(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	index := ns.readFile("/public/index.html")
	c.Contains(index, "/other-page/")
}

func TestContentFingerprint(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
		Fingerprint: func(c *Content) bool {
			return filepath.Ext(c.URL.Path) == ".css"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	allCSS := cont.GetPage("/all.css")
	c.NotLen(allCSS.Fingerprint, 0)
	c.Contains(allCSS.URL.Path, allCSS.Fingerprint)

	index := ns.readFile("/public/index.html")
	c.Contains(index, allCSS.URL.Path)
}

func TestContentAlias(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

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

	ns := newTestNS(c, nil)
	defer ns.clean()

	cfg := Config{
		Handler: variantHandler,
		Entries: []string{"/"},
		Output:  ns.path("/public"),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	c.Equal(cont.Get("/people/?who=bob").URL.Path, "/people/bob.html")
	c.Equal(cont.Get("/people/?who=alice").URL.Path, "/people/alice.html")

	index := ns.readFile("/public/index.html")
	c.Contains(index, "people/bob.html")
	c.Contains(index, "people/alice.html")

	c.Equal(ns.readFile("/public/people/bob.html"), "bob is a person")
	c.Equal(ns.readFile("/public/people/alice.html"), "alice is cool")
}

func TestContentVariantFingerprint(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

	cfg := Config{
		Handler: variantHandler,
		Entries: []string{"/"},
		Output:  ns.path("/public"),
		Fingerprint: func(c *Content) bool {
			return c.URL.Path != "/"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	bob := cont.Get("/people/?who=bob")
	c.NotContains(bob.URL.Path, "bob.html")
	c.NotEqual(bob.Fingerprint, "")

	alice := cont.Get("/people/?who=alice")
	c.NotContains(alice.URL.Path, "alice.html")
	c.NotEqual(alice.Fingerprint, "")

	index := ns.readFile("/public/index.html")
	c.Contains(index, bob.Fingerprint)
	c.Contains(index, alice.Fingerprint)

	c.Equal(
		ns.readFile(filepath.Join("/public", bob.URL.Path)),
		"bob is a person")
	c.Equal(
		ns.readFile(filepath.Join("/public", alice.URL.Path)),
		"alice is cool")
}

func TestContentURLFragment(t *testing.T) {
	c := check.New(t)

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

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	index := ns.readFile("/public/index.html")
	for _, link := range links {
		c.Contains(index, link)
	}
}

func TestContentBodyChanged(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, map[string][]byte{
		"/public/index.html":        []byte("index"),
		"/public/page/1/index.html": []byte("page 0"),
		"/public/page/2/index.html": []byte("totally different"),
	})
	defer ns.clean()

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
		Output: ns.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	c.Equal(ns.readFile("/public/index.html"), "index")
	c.Equal(ns.readFile("/public/page/1/index.html"), "page 1")
	c.Equal(ns.readFile("/public/page/2/index.html"), "page 2")
}

func TestContentOutputCollision(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
		Fingerprint: func(c *Content) bool {
			return strings.Contains(c.URL.Path, "img.gif")
		},
	}

	_, err = Crawl(cfg)
	c.Must.NotNil(err)

	c.Len(err, 1)
	for _, errs := range err.(Error) {
		c.Equal(
			errs[0].(AlreadyClaimedError).Path,
			ns.path(filepath.Join("/public/", gifPath)))
		c.Contains(errs[0].Error(), "already claimed by")
	}
}

func TestContentRedirectLoop(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
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
	ns.dumpTree()

	c.True(called)
}

func TestContentTooManyRedirects(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
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
	ns.dumpTree()

	c.True(called)
}

func TestContentErrors(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

	tests := []struct {
		name    string
		errPath string
		cfg     Config
	}{
		{
			name:    "Basic404",
			errPath: "/",
			cfg: Config{
				Handler: http.HandlerFunc(http.NotFound),
				Entries: []string{"/"},
				Output:  ns.path("/public"),
			},
		},
		{
			name:    "InvalidContentType",
			errPath: "/",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/": stringHandler{
						contType: "invalid; ======",
					},
				}),
				Entries: []string{"/"},
				Output:  ns.path("/public"),
			},
		},
		{
			name:    "ContentTypeMismatch",
			errPath: "/page.html",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/page.html": stringHandler{
						contType: gifType,
					},
				}),
				Entries: []string{"/page.html"},
				Output:  ns.path("/public"),
			},
		},
		{
			name:    "UnknownContentTypeExtension",
			errPath: "/page.not-an-ext",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/page.not-an-ext": stringHandler{
						contType: gifType,
					},
				}),
				Entries: []string{"/page.not-an-ext"},
				Output:  ns.path("/public"),
			},
		},
		{
			name:    "ServeNonExistent",
			errPath: "/all.css",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/all.css": http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							ServeFile(w, r, ns.path("doesnt-exist.css"))
						}),
				}),
				Entries: []string{"/all.css"},
				Output:  ns.path("/public"),
			},
		},
		{
			name:    "FingerprintNonExistent",
			errPath: "/img.gif",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/img.gif": http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							ServeFile(w, r, ns.path("doesnt-exist.gif"))
						}),
				}),
				Entries:     []string{"/img.gif"},
				Output:      ns.path("/public"),
				Fingerprint: func(c *Content) bool { return true },
			},
		},
		{
			name:    "InvalidVariantURL",
			errPath: "/",
			cfg: Config{
				Handler: http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						Variant(w, "://")
					}),
				Entries: []string{"/"},
				Output:  ns.path("/public"),
			},
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.name, func(c *check.C) {
			_, err := Crawl(test.cfg)
			c.Log(err)
			c.Contains(err, test.errPath)
		})
	}
}
