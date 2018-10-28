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

	"github.com/thatguystone/acrylic/internal/testutil"
	"github.com/thatguystone/cog/check"
)

func TestPageAddIndexSanity(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, nil)
	defer tmp.Remove()

	_, err := Crawl(
		mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `index`,
			},
		}),
		Output(tmp.Path("/public")))
	c.Nil(err)
	tmp.DumpTree()

	c.Equal(tmp.ReadFile("/public/index.html"), `index`)
}

func TestPageFingerprint(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, nil)
	defer tmp.Remove()

	site, err := Crawl(
		mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<link href="all.css" rel="stylesheet">`,
			},
			"/all.css": stringHandler{
				contType: cssType,
				body:     `body { background: #000; }`,
			},
		}),
		Output(tmp.Path("/public")),
		Fingerprint(func(u *url.URL, mediaType string) bool {
			return filepath.Ext(u.Path) == ".css"
		}))
	c.Must.Nil(err)
	tmp.DumpTree()

	allCSS := site.GetPage("/all.css")
	c.NotLen(allCSS.Fingerprint, 0)
	c.Contains(allCSS.URL.Path, allCSS.Fingerprint)

	index := tmp.ReadFile("/public/index.html")
	c.Contains(index, allCSS.URL.Path)
}

func TestPageAlias(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, nil)
	defer tmp.Remove()

	site, err := Crawl(
		mux(map[string]http.Handler{
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
		Output(tmp.Path("/public")))
	c.Must.Nil(err)
	tmp.DumpTree()

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

	tmp := testutil.NewTmpDir(c, map[string]string{
		"/public/index.html/is/a/dir/index.html": `not index`,
		"/public/about.html":                     `not about`,
		"/public/img.gif":                        `not a gif`,
	})
	defer tmp.Remove()

	_, err := Crawl(
		mux(map[string]http.Handler{
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
				contType: testutil.GifType,
				body:     string(testutil.GifBin),
			},
		}),
		Output(tmp.Path("/public")))
	c.Nil(err)
	tmp.DumpTree()

	c.Equal(tmp.ReadFile("/public/about.html"), `about`)
	c.Equal(tmp.ReadFile("/public/img.gif"), string(testutil.GifBin))
}

func TestPageServeFileSymlinks(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, map[string]string{
		"/stuff":                     `stuff`,
		"/stuff.txt":                 `stuff`,
		"/stuff.css":                 ` body { } `,
		"/public/stuff/is/a/dir":     `not stuff`,
		"/public/stuff.txt/is/a/dir": `not stuff`,
		"/public/stuff.css/is/a/dir": `not stuff`,
	})
	defer tmp.Remove()

	_, err := Crawl(
		mux(map[string]http.Handler{
			"/stuff": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, tmp.Path("/stuff"))
				}),
			"/stuff.txt": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, tmp.Path("/stuff.txt"))
				}),
			"/stuff.css": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, tmp.Path("/stuff.css"))
				}),
		}),
		Entry(
			&url.URL{Path: "/stuff"},
			&url.URL{Path: "/stuff.txt"},
			&url.URL{Path: "/stuff.css"}),
		Output(tmp.Path("/public")))
	c.Must.Nil(err)
	tmp.DumpTree()

	symSrc, err := os.Readlink(tmp.Path("/public/stuff"))
	c.Must.Nil(err)
	c.Equal(symSrc, tmp.Path("/stuff"))
	c.Equal(tmp.ReadFile("/public/stuff"), `stuff`)

	symSrc, err = os.Readlink(tmp.Path("/public/stuff.txt"))
	c.Must.Nil(err)
	c.Equal(symSrc, tmp.Path("/stuff.txt"))
	c.Equal(tmp.ReadFile("/public/stuff.txt"), `stuff`)

	// Files with transforms shouldn't be linked
	symSrc, err = os.Readlink(tmp.Path("/public/stuff.css"))
	c.NotNil(err)
	c.Equal(symSrc, "")
	c.Equal(tmp.ReadFile("/public/stuff.css"), `body{}`)
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

	tmp := testutil.NewTmpDir(c, nil)
	defer tmp.Remove()

	site, err := Crawl(variantHandler, Output(tmp.Path("/public")))
	c.Must.Nil(err)
	tmp.DumpTree()

	c.Equal(
		site.Get(&url.URL{Path: "/people/", RawQuery: "who=bob"}).URL.Path,
		"/people/bob.html")
	c.Equal(
		site.Get(&url.URL{Path: "/people/", RawQuery: "who=alice"}).URL.Path,
		"/people/alice.html")

	index := tmp.ReadFile("/public/index.html")
	c.Contains(index, "people/bob.html")
	c.Contains(index, "people/alice.html")

	c.Equal(tmp.ReadFile("/public/people/bob.html"), "bob is a person")
	c.Equal(tmp.ReadFile("/public/people/alice.html"), "alice is cool")
}

func TestPageVariantFingerprint(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, nil)
	defer tmp.Remove()

	site, err := Crawl(
		variantHandler,
		Output(tmp.Path("/public")),
		Fingerprint(func(u *url.URL, mediaType string) bool {
			return u.Path != "/"
		}))
	c.Must.Nil(err)
	tmp.DumpTree()

	bob := site.Get(&url.URL{Path: "/people/", RawQuery: "who=bob"})
	c.NotContains(bob.URL.Path, "bob.html")
	c.NotEqual(bob.Fingerprint, "")

	alice := site.Get(&url.URL{Path: "/people/", RawQuery: "who=alice"})
	c.NotContains(alice.URL.Path, "alice.html")
	c.NotEqual(alice.Fingerprint, "")

	index := tmp.ReadFile("/public/index.html")
	c.Contains(index, bob.Fingerprint)
	c.Contains(index, alice.Fingerprint)

	c.Equal(
		tmp.ReadFile(filepath.Join("/public", bob.URL.Path)),
		"bob is a person")
	c.Equal(
		tmp.ReadFile(filepath.Join("/public", alice.URL.Path)),
		"alice is cool")
}

func TestPageURLFragments(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, nil)
	defer tmp.Remove()

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

	_, err := Crawl(
		mux(map[string]http.Handler{
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
		Output(tmp.Path("/public")))
	c.Must.Nil(err)
	tmp.DumpTree()

	index := tmp.ReadFile("/public/index.html")
	for _, link := range links {
		c.Contains(index, link)
	}
}

func TestPageRedirectBasic(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, nil)
	defer tmp.Remove()

	site, err := Crawl(
		mux(map[string]http.Handler{
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
		Output(tmp.Path("/public")))
	c.Nil(err)
	tmp.DumpTree()

	index := tmp.ReadFile("/public/index.html")
	c.Contains(index, "/f/")
	c.NotContains(index, "/r/")

	c.Equal(
		site.Get(&url.URL{Path: "/r/"}).FollowRedirects().URL.String(),
		"/f/")
}

func TestPageBodyChanged(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, map[string]string{
		"/public/index.html":        `index`,
		"/public/page/1/index.html": `page 0`,
		"/public/page/2/index.html": `totally different`,
	})
	defer tmp.Remove()

	_, err := Crawl(
		mux(map[string]http.Handler{
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
		Entry(
			&url.URL{Path: "/"},
			&url.URL{Path: "/page/1/"},
			&url.URL{Path: "/page/2/"}),
		Output(tmp.Path("/public")))
	c.Must.Nil(err)
	tmp.DumpTree()

	c.Equal(tmp.ReadFile("/public/index.html"), "index")
	c.Equal(tmp.ReadFile("/public/page/1/index.html"), "page 1")
	c.Equal(tmp.ReadFile("/public/page/2/index.html"), "page 2")
}

func TestPageErrors(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		name string
		h    http.Handler
		opts []Option
		err  SiteError
	}{
		{
			name: "Basic404",
			h:    http.HandlerFunc(http.NotFound),
			err: SiteError{
				"/": {
					ResponseError{
						Status: http.StatusNotFound,
						Body:   []byte(`404 page not found`),
					},
				},
			},
		},
		{
			name: "TransformError",
			h: mux(map[string]http.Handler{
				"/": stringHandler{
					contType: htmlType,
				},
			}),
			opts: []Option{
				Transforms(map[string][]Transform{
					htmlType: {func(lr LinkResolver, b []byte) ([]byte, error) {
						return nil, errors.New("transform failed")
					}},
				}),
			},
			err: SiteError{
				"/": {
					errors.New("transform failed"),
				},
			},
		},
		{
			name: "InvalidContentType",
			h: mux(map[string]http.Handler{
				"/": stringHandler{
					contType: "invalid; ======",
				},
			}),
			err: SiteError{
				"/": {errors.New("mime: invalid media parameter")},
			},
		},
		{
			name: "ContentTypeMismatch",
			h: mux(map[string]http.Handler{
				"/page.html": stringHandler{
					contType: testutil.GifType,
				},
			}),
			opts: []Option{
				Entry(&url.URL{Path: "/page.html"}),
			},
			err: SiteError{
				"/page.html": {
					MimeTypeMismatchError{
						Ext:          ".html",
						Guess:        htmlType,
						FromResponse: testutil.GifType,
					},
				},
			},
		},
		{
			name: "UnknownContentTypeExtension",
			h: mux(map[string]http.Handler{
				"/page.not-an-ext": stringHandler{
					contType: testutil.GifType,
				},
			}),
			opts: []Option{
				Entry(&url.URL{Path: "/page.not-an-ext"}),
			},
			err: SiteError{
				"/page.not-an-ext": {
					MimeTypeMismatchError{
						Ext:          ".not-an-ext",
						Guess:        DefaultType,
						FromResponse: testutil.GifType,
					},
				},
			},
		},
		{
			name: "TransformNonExistentServeFile",
			h: mux(map[string]http.Handler{
				"/all.css": http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						ServeFile(w, r, "/doesnt-exist.css")
					}),
			}),
			opts: []Option{
				Entry(&url.URL{Path: "/all.css"}),
			},
			err: SiteError{
				"/all.css": {ResponseError{
					Status: http.StatusNotFound,
					Body:   []byte(`file "/doesnt-exist.css" does not exist`),
				}},
			},
		},
		{
			name: "FingerprintNonExistent",
			h: mux(map[string]http.Handler{
				"/img.gif": http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						ServeFile(w, r, "/doesnt-exist.gif")
					}),
			}),
			opts: []Option{
				Entry(&url.URL{Path: "/img.gif"}),
				Fingerprint(func(u *url.URL, mediaType string) bool {
					return true
				}),
			},
			err: SiteError{
				"/img.gif": {ResponseError{
					Status: http.StatusNotFound,
					Body:   []byte(`file "/doesnt-exist.gif" does not exist`),
				}},
			},
		},
		{
			name: "InvalidHref",
			h: mux(map[string]http.Handler{
				"/": stringHandler{
					contType: htmlType,
					body:     `<a href="://"></a>`,
				},
			}),
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
			h: http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					Variant(w, "://")
				}),
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
			h: http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Location", "://")
					w.WriteHeader(http.StatusFound)
				}),
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
			h: mux(map[string]http.Handler{
				"/": stringHandler{
					contType: htmlType,
					body:     `<a href="/infinite/">inf</a>`,
				},
				"/infinite/": http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						http.Redirect(w, r, "/infinite/", http.StatusFound)
					}),
			}),
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
			h: mux(map[string]http.Handler{
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
			tmp := testutil.NewTmpDir(c, nil)
			defer tmp.Remove()

			var opts []Option
			opts = append(opts, test.opts...)
			opts = append(opts, Output(tmp.Path("/public")))

			_, err := Crawl(test.h, opts...)
			c.Equal(err, test.err)
			if err != nil {
				c.Equal(err.Error(), test.err.Error())
			}
		})
	}
}
