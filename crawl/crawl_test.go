package crawl

import (
	"io"
	"net/http"
	"path/filepath"
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

const gifType = "image/gif"

var gifBin = []byte{
	0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
	0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
	0x00, 0x3b,
}

func TestCrawlBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<link href="all.css" rel="stylesheet">` +
					`<style>.body { background: url(body.gif); }</style>` +
					`<a href="dir/dir/about.html?args=1#to-1" style="background: url('link.gif')">about 1</a>` +
					`<a href="dir/dir/about.html?args=2#to-2">about 2</a>` +
					`root`,
			},
			"/dir/dir/about.html": stringHandler{
				contType: htmlType,
				body: `<link href="../../all.css" rel="stylesheet">about` +
					`<a href="../../">index</a>`,
			},
			"/all.css": stringHandler{
				contType: cssType,
				body: `@import "print.css" print;` +
					` .logo { background: url("logo.gif"); }`,
			},
			"/print.css": stringHandler{
				contType: cssType,
				body:     `.print {}`,
			},
			"/logo.gif": stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
			"/link.gif": stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
			"/body.gif": stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
		}),
		Entries: []string{
			"/",
		},
		Output: fs.Path("."),
		Fingerprint: func(c *Content) bool {
			return filepath.Ext(c.Src.Path) == ".css"
		},
	}

	_, err := Crawl(cfg)
	c.Nil(err)

	fs.DumpTree(".")

	c.Equal(fs.SReadFile("index.html"), "")
	c.Equal(fs.SReadFile("dir/dir/about.html"), "about")

	// fmt.Println(contents["/print.css"].Dst)
}

func TestCrawlFingerprint(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<link href="all.css" rel="stylesheet">` +
					`<link href="all.css?t=light" rel="stylesheet">` +
					`<link href="all.css?t=dark" rel="stylesheet">` +
					`<link href="all.css?t=redir" rel="stylesheet">` +
					`<script src="all.js?q=1">`,
			},
			"/all.css": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					switch r.FormValue("t") {
					case "dark":
						stringHandler{
							contType: cssType,
							body:     `body { background: #000; }`,
						}.ServeHTTP(w, r)

					case "redir":
						http.Redirect(w, r, "redir.css", http.StatusFound)

					default:
						stringHandler{
							contType: cssType,
							body:     `body { background: #fff; }`,
						}.ServeHTTP(w, r)
					}
				}),
			"/redir.css": stringHandler{
				contType: cssType,
				body:     `.body { }`,
			},
			"/all.js": stringHandler{
				contType: jsType,
				body:     `function() {}`,
			},
		}),
		Entries: []string{
			"/",
		},
		Output: fs.Path("."),
		Fingerprint: func(c *Content) bool {
			switch filepath.Ext(c.Src.Path) {
			case ".css", ".js":
				return true
			default:
				return false
			}
		},
	}

	_, err := Crawl(cfg)
	c.Nil(err)

	fs.DumpTree(".")
	c.Equal(fs.SReadFile("index.html"), "")
}

func TestCrawlVariant(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<link href="all.css" rel="stylesheet">` +
					`<link href="all.css?c=dark" rel="stylesheet">` +
					`<link href="all.css?c=with-query" rel="stylesheet">`,
			},
			"/all.css": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					switch r.FormValue("c") {
					case "dark":
						Variant(w, "all.dark.css")
						stringHandler{
							contType: cssType,
							body:     `body { background: #000; }`,
						}.ServeHTTP(w, r)

					case "with-query":
						Variant(w, "all.derp.css?c=with-query")
						stringHandler{
							contType: cssType,
							body:     `body { background: #333; }`,
						}.ServeHTTP(w, r)

					default:
						stringHandler{
							contType: cssType,
							body:     `body { background: #fff; }`,
						}.ServeHTTP(w, r)
					}
				}),
		}),
		Entries: []string{
			"/",
		},
		Output: fs.Path("."),
		Fingerprint: func(c *Content) bool {
			return filepath.Ext(c.Src.Path) == ".css"
		},
	}

	_, err := Crawl(cfg)
	c.Nil(err)

	fs.DumpTree(".")
	c.Equal(fs.SReadFile("index.html"), "")
}

func TestCrawlURLFragment(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body: `` +
					`<a href="//google.com#frag-1">link 1</a>` +
					`<a href="//google.com#frag-2">link 2</a>`,
			},
		}),
		Entries: []string{
			"/",
		},
		Output: fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Nil(err)

	fs.DumpTree(".")
	c.Equal(fs.SReadFile("index.html"), "")
}

func TestCrawlContentError(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<link href="all.css" rel="stylesheet">`,
			},
			"/all.css": http.HandlerFunc(http.NotFound),
		}),
		Entries: []string{
			"/",
		},
		Output: fs.Path("."),
		Fingerprint: func(c *Content) bool {
			return filepath.Ext(c.Src.Path) == ".css"
		},
	}

	_, err := Crawl(cfg)
	c.Nil(err)

	fs.DumpTree(".")
	c.Equal(fs.SReadFile("index.html"), "")
}
