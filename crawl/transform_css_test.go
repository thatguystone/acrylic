package crawl

import (
	"net/http"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestTransformCSSBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/all.css": stringHandler{
				contType: cssType,
				body: `@import "/r/";` +
					`body { background: url(/r/); }`,
			},
			"/r/": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Redirect(w, r, "/f/", http.StatusFound)
				}),
			"/f/": stringHandler{
				contType: htmlType,
				body:     `redirected`,
			},
		}),
		Entries: []string{"/all.css"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	all := fs.SReadFile("all.css")
	c.NotContains(all, "/r/")
	c.Contains(all, "/f/")
}

func TestTransformCSSError(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/all.css": stringHandler{
				contType: cssType,
				body:     `body { derp`,
			},
		}),
		Entries: []string{"/all.css"},
		Output:  fs.Path("."),
	}

	_, err := Crawl(cfg)
	c.Must.NotNil(err)
	c.Contains(err, "/all.css")
}
