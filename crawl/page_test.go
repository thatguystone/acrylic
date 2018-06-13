package crawl

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
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

	site, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	allCSS := site.GetPage("/all.css")
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

	site, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	p1 := site.Get(&url.URL{Path: "/page/", RawQuery: "p=1"})
	c.Equal(p1.URL.RawQuery, "p=1")

	p2 := site.Get(&url.URL{Path: "/page/", RawQuery: "p=2"})
	c.Equal(p2.URL.RawQuery, "p=2")

	c.Equal(p1.URL.Path, p2.URL.Path)
	c.Equal(p1.OutputPath, p2.OutputPath)
	c.Equal(p1.Fingerprint, p2.Fingerprint)
}

func TestPageOverwriteExistingOutputs(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, map[string]string{
		"/public/index.html/is/a/dir/index.html": `not index`,
		"/public/about.html":                     `not about`,
		"/public/img.gif":                        `not a gif`,
	})
	defer tmp.remove()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<img src="/img.gif">` +
					`<a href="/about.html">about</a>`,
			},
			"/about.html": stringHandler{
				contType: htmlType,
				body:     `about`,
			},
			"/img.gif": stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
		}),
		Output: tmp.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Nil(err)

	c.Equal(tmp.readFile("/public/about.html"), `about`)
	c.Equal(tmp.readFile("/public/img.gif"), string(gifBin))
}

func TestPageServeFileSymlinks(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, map[string]string{
		"/stuff":                     `stuff`,
		"/stuff.txt":                 `stuff`,
		"/stuff.css":                 ` body { } `,
		"/public/stuff/is/a/dir":     `not stuff`,
		"/public/stuff.txt/is/a/dir": `not stuff`,
		"/public/stuff.css/is/a/dir": `not stuff`,
	})
	defer tmp.remove()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/stuff": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, tmp.path("/stuff"))
				}),
			"/stuff.txt": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, tmp.path("/stuff.txt"))
				}),
			"/stuff.css": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, tmp.path("/stuff.css"))
				}),
		}),
		Entries: []*url.URL{
			{Path: "/stuff"},
			{Path: "/stuff.txt"},
			{Path: "/stuff.css"},
		},
		Output: tmp.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	symSrc, err := os.Readlink(tmp.path("/public/stuff"))
	c.Must.Nil(err)
	c.Equal(symSrc, tmp.path("/stuff"))
	c.Equal(tmp.readFile("/public/stuff"), `stuff`)

	symSrc, err = os.Readlink(tmp.path("/public/stuff.txt"))
	c.Must.Nil(err)
	c.Equal(symSrc, tmp.path("/stuff.txt"))
	c.Equal(tmp.readFile("/public/stuff.txt"), `stuff`)

	// Files with transforms shouldn't be linked
	symSrc, err = os.Readlink(tmp.path("/public/stuff.css"))
	c.NotNil(err)
	c.Equal(symSrc, "")
	c.Equal(tmp.readFile("/public/stuff.css"), `body{}`)
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

	site, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	c.Equal(
		site.Get(&url.URL{Path: "/people/", RawQuery: "who=bob"}).URL.Path,
		"/people/bob.html")
	c.Equal(
		site.Get(&url.URL{Path: "/people/", RawQuery: "who=alice"}).URL.Path,
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

	site, err := Crawl(cfg)
	c.Must.Nil(err)
	tmp.dumpTree()

	bob := site.Get(&url.URL{Path: "/people/", RawQuery: "who=bob"})
	c.NotContains(bob.URL.Path, "bob.html")
	c.NotEqual(bob.Fingerprint, "")

	alice := site.Get(&url.URL{Path: "/people/", RawQuery: "who=alice"})
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

func TestPageRedirectBasic(t *testing.T) {
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

	site, err := Crawl(cfg)
	c.Nil(err)

	index := tmp.readFile("/public/index.html")
	c.Contains(index, "/f/")
	c.NotContains(index, "/r/")

	c.Equal(
		site.Get(&url.URL{Path: "/r/"}).FollowRedirects().URL.String(),
		"/f/")
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

	_, cssOpenErr := os.Open("/doesnt-exist.css")
	_, gifOpenErr := os.Open("/doesnt-exist.gif")

	tests := []struct {
		name string
		cfg  Config
		err  SiteError
	}{
		{
			name: "Basic404",
			cfg: Config{
				Handler: http.HandlerFunc(http.NotFound),
			},
			err: SiteError{
				"/": {
					ResponseError{
						Status: http.StatusNotFound,
						Body:   []byte(`404 page not found` + "\n"),
					},
				},
			},
		},
		{
			name: "TransformError",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/all.css": stringHandler{
						contType: cssType,
						body:     `body { invalid`,
					},
				}),
				Entries: []*url.URL{
					&url.URL{Path: "/all.css"},
				},
			},
			err: SiteError{
				"/all.css": {
					errors.New("unexpected token in declaration, expected colon token"),
				},
			},
		},
		{
			name: "InvalidContentType",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/": stringHandler{
						contType: "invalid; ======",
					},
				}),
			},
			err: SiteError{
				"/": {errors.New("mime: invalid media parameter")},
			},
		},
		{
			name: "ContentTypeMismatch",
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
			err: SiteError{
				"/page.html": {
					MimeTypeMismatchError{
						Ext:          ".html",
						Guess:        htmlType,
						FromResponse: gifType,
					},
				},
			},
		},
		{
			name: "UnknownContentTypeExtension",
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
			err: SiteError{
				"/page.not-an-ext": {
					MimeTypeMismatchError{
						Ext:          ".not-an-ext",
						Guess:        DefaultType,
						FromResponse: gifType,
					},
				},
			},
		},
		{
			name: "TransformNonExistentServeFile",
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
			err: SiteError{
				"/all.css": {cssOpenErr},
			},
		},
		{
			name: "FingerprintNonExistent",
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
				Fingerprint: func(u *url.URL, mediaType string) bool {
					return true
				},
			},
			err: SiteError{
				"/img.gif": {gifOpenErr},
			},
		},
		{
			name: "InvalidHref",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/": stringHandler{
						contType: htmlType,
						body:     `<a href="://"></a>`,
					},
				}),
			},
			err: SiteError{
				"/": {
					&url.Error{
						Op:  "parse",
						URL: "://",
						Err: errors.New("missing protocol scheme"),
					},
				},
			},
		},
		{
			name: "InvalidVariantURL",
			cfg: Config{
				Handler: http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						Variant(w, "://")
					}),
			},
			err: SiteError{
				"/": {
					&url.Error{
						Op:  "parse",
						URL: "://",
						Err: errors.New("missing protocol scheme"),
					},
				},
			},
		},
		{
			name: "InvalidRedirectURL",
			cfg: Config{
				Handler: http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Location", "://")
						w.WriteHeader(http.StatusFound)
					}),
			},
			err: SiteError{
				"/": {
					&url.Error{
						Op:  "parse",
						URL: "://",
						Err: errors.New("missing protocol scheme"),
					},
				},
			},
		},
		{
			name: "RedirectLoop",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/": stringHandler{
						contType: htmlType,
						body:     `<a href="/infinite/">inf</a>`,
					},
					"/infinite/": http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							http.Redirect(w, r, "/infinite/", http.StatusFound)
						}),
				}),
			},
			err: SiteError{
				"/": {
					RedirectLoopError{
						Start: "/infinite/",
						End:   "/infinite/",
					},
				},
			},
		},
		{
			name: "TooManyRedirects",
			cfg: Config{
				Handler: mux(map[string]http.Handler{
					"/": stringHandler{
						contType: htmlType,
						body:     `<a href="/deep/">deep</a>`,
					},
					"/deep/": http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							i, _ := strconv.Atoi(path.Base(r.URL.Path))

							if i > maxRedirects {
								w.Header().Set("Content-Type", DefaultType)
								io.WriteString(w, "done")
							} else {
								to := fmt.Sprintf("/deep/%d", i+1)
								http.Redirect(w, r, to, http.StatusFound)
							}
						}),
				}),
			},
			err: SiteError{
				"/": {
					TooManyRedirectsError{
						Start: "/deep/",
						End:   "/deep/25",
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.name, func(c *check.C) {
			tmp := newTmpDir(c, nil)
			defer tmp.remove()

			test.cfg.Output = tmp.path("/public")

			_, err := Crawl(test.cfg)
			c.Equal(err, test.err)
			if err != nil {
				c.Equal(err.Error(), test.err.Error())
			}
		})
	}
}
