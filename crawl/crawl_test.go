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
				body:     `<link href="../../all.css" rel="stylesheet">about`,
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
				contType: "image/gif",
				body:     string(gifBin),
			},
			"/link.gif": stringHandler{
				contType: "image/gif",
				body:     string(gifBin),
			},
			"/body.gif": stringHandler{
				contType: "image/gif",
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
		Links: AbsoluteLinks,
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
					`<link href="all.css?q=1" rel="stylesheet">` +
					`<link href="all.css?q=2" rel="stylesheet">` +
					`<link href="all.css?q=3" rel="stylesheet">` +
					`<script src="all.js?q=1">`,
			},
			"/all.css": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.FormValue("q") == "3" {
					http.Redirect(w, r, "redir.css", http.StatusFound)
					return
				}

				stringHandler{
					contType: cssType,
					body:     `.logo { }`,
				}.ServeHTTP(w, r)
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
