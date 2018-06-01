package crawl

import (
	"fmt"
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

func TestCrawlInlineStyles(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<div style="background: url(img.gif);"></div>`,
			},

			"/img.gif": stringHandler{
				contType: gifType,
				body:     string(gifBin),
			},
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	fs.FileExists("img.gif")
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
			return filepath.Ext(c.Src.Path) == ".css"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	allCSS := cont.GetPage("/all.css")
	c.NotLen(allCSS.Fingerprint, 0)
	c.Contains(allCSS.Dst.Path, allCSS.Fingerprint)

	index := fs.SReadFile("index.html")
	c.Contains(index, allCSS.Dst.Path)
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
		}),
		Entries: []string{"/"},
		Output:  fs.Path("."),
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	c.Equal(cont.Get("/people/?who=bob").Dst.Path, "/people/bob.html")
	c.Equal(cont.Get("/people/?who=alice").Dst.Path, "/people/alice.html")

	index := fs.SReadFile("index.html")
	c.Contains(index, "people/bob.html")
	c.Contains(index, "people/alice.html")

	c.Equal(fs.SReadFile("people/bob.html"), "bob is a person")
	c.Equal(fs.SReadFile("people/alice.html"), "alice is cool")
}

func TestCrawlVariantFingerprint(t *testing.T) {

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
