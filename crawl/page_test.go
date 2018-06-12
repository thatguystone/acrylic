package crawl

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestPageAddIndexSanity(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, nil)
	defer tmp.remove()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `index`,
			},
		}),
		Output: tmp.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Nil(err)
	tmp.dumpTree()

	c.Equal(tmp.readFile("/public/index.html"), `index`)
}

func TestPageFingerprint(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, nil)
	defer tmp.remove()

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
		Output: tmp.path("/public"),
		Fingerprint: func(u *url.URL, mediaType string) bool {
			return filepath.Ext(u.Path) == ".css"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	allCSS := cont.GetPage("/all.css")
	c.NotLen(allCSS.Fingerprint, 0)
	c.Contains(allCSS.URL.Path, allCSS.Fingerprint)

	index := tmp.readFile("/public/index.html")
	c.Contains(index, allCSS.URL.Path)
}

func TestPageAlias(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, nil)
	defer tmp.remove()

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
		Output: tmp.path("/public"),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	p1 := cont.Get(&url.URL{Path: "/page/", RawQuery: "p=1"})
	c.Equal(p1.URL.RawQuery, "p=1")

	p2 := cont.Get(&url.URL{Path: "/page/", RawQuery: "p=2"})
	c.Equal(p2.URL.RawQuery, "p=2")

	c.Equal(p1.URL.Path, p2.URL.Path)
	c.Equal(p1.OutputPath, p2.OutputPath)
	c.Equal(p1.Fingerprint, p2.Fingerprint)
}

func TestPageServeFile(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, map[string]string{
		"/stuff.txt": `stuff`,
	})
	defer tmp.remove()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/stuff.txt": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, tmp.path("/stuff.txt"))
				}),
		}),
		Entries: []*url.URL{
			{Path: "/stuff.txt"},
		},
		Output: tmp.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	symSrc, err := os.Readlink(tmp.path("/public/stuff.txt"))
	c.Nil(err)
	c.Equal(symSrc, tmp.path("/stuff.txt"))
	c.Equal(tmp.readFile("/public/stuff.txt"), `stuff`)
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

func TestPageVariantBasic(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, nil)
	defer tmp.remove()

	cfg := Config{
		Handler: variantHandler,
		Output:  tmp.path("/public"),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	c.Equal(
		cont.Get(&url.URL{Path: "/people/", RawQuery: "who=bob"}).URL.Path,
		"/people/bob.html")
	c.Equal(
		cont.Get(&url.URL{Path: "/people/", RawQuery: "who=alice"}).URL.Path,
		"/people/alice.html")

	index := tmp.readFile("/public/index.html")
	c.Contains(index, "people/bob.html")
	c.Contains(index, "people/alice.html")

	c.Equal(tmp.readFile("/public/people/bob.html"), "bob is a person")
	c.Equal(tmp.readFile("/public/people/alice.html"), "alice is cool")
}

func TestPageVariantFingerprint(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, nil)
	defer tmp.remove()

	cfg := Config{
		Handler: variantHandler,
		Output:  tmp.path("/public"),
		Fingerprint: func(u *url.URL, mediaType string) bool {
			return u.Path != "/"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	bob := cont.Get(&url.URL{Path: "/people/", RawQuery: "who=bob"})
	c.NotContains(bob.URL.Path, "bob.html")
	c.NotEqual(bob.Fingerprint, "")

	alice := cont.Get(&url.URL{Path: "/people/", RawQuery: "who=alice"})
	c.NotContains(alice.URL.Path, "alice.html")
	c.NotEqual(alice.Fingerprint, "")

	index := tmp.readFile("/public/index.html")
	c.Contains(index, bob.Fingerprint)
	c.Contains(index, alice.Fingerprint)

	c.Equal(
		tmp.readFile(filepath.Join("/public", bob.URL.Path)),
		"bob is a person")
	c.Equal(
		tmp.readFile(filepath.Join("/public", alice.URL.Path)),
		"alice is cool")
}

func TestPageURLFragments(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, nil)
	defer tmp.remove()

	links := []string{
		"//google.com#frag-1",
		"//google.com#frag-2",
		"//google.com/#frag-2",
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
			"/about.html": stringHandler{
				contType: htmlType,
				body:     `about`,
			},
			"/page/": stringHandler{
				contType: htmlType,
				body:     `<div id="frag"></div>`,
			},
		}),
		Output: tmp.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	index := tmp.readFile("/public/index.html")
	for _, link := range links {
		c.Contains(index, link)
	}
}

func TestPageRedirect(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, nil)
	defer tmp.remove()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<a href="/r/">link</a>`,
			},
			"/r/": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, "/f/", http.StatusFound)
				}),
			"/f/": stringHandler{
				contType: htmlType,
				body:     `file`,
			},
		}),
		Output: tmp.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Nil(err)

	index := tmp.readFile("/public/index.html")
	c.Contains(index, "/f/")
	c.NotContains(index, "/r/")
}

func TestPageBodyChanged(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, map[string]string{
		"/public/index.html":        `index`,
		"/public/page/1/index.html": `page 0`,
		"/public/page/2/index.html": `totally different`,
	})
	defer tmp.remove()

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
		Entries: []*url.URL{
			&url.URL{Path: "/"},
			&url.URL{Path: "/page/1/"},
			&url.URL{Path: "/page/2/"},
		},
		Output: tmp.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	c.Equal(tmp.readFile("/public/index.html"), "index")
	c.Equal(tmp.readFile("/public/page/1/index.html"), "page 1")
	c.Equal(tmp.readFile("/public/page/2/index.html"), "page 2")
}

func TestPageErrors(t *testing.T) {
	c := check.New(t)

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
				Entries: []*url.URL{
					&url.URL{Path: "/page.html"},
				},
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
				Entries: []*url.URL{
					&url.URL{Path: "/page.not-an-ext"},
				},
			},
		},
		{
			name:    "ServeNonExistent",
			errPath: "/all.css",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/all.css": http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							ServeFile(w, r, "/doesnt-exist.css")
						}),
				}),
				Entries: []*url.URL{
					&url.URL{Path: "/all.css"},
				},
			},
		},
		{
			name:    "FingerprintNonExistent",
			errPath: "/img.gif",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/img.gif": http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							ServeFile(w, r, "/doesnt-exist.gif")
						}),
				}),
				Entries: []*url.URL{
					&url.URL{Path: "/img.gif"},
				},
				Fingerprint: func(u *url.URL, mediaType string) bool { return true },
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
			},
		},
	}

	for _, test := range tests {
		c.Run(test.name, func(c *check.C) {
			tmp := newTmpDir(c, nil)
			defer tmp.remove()

			test.cfg.Output = tmp.path("/public")

			_, err := Crawl(test.cfg)
			c.Log(err)
			c.Contains(err, test.errPath)
		})
	}
}

func TestPageOSErrors(t *testing.T) {
	c := check.New(t)

	c.UntilNil(100, func() error {
		return nil
	})
}
