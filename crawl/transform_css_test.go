package crawl

import (
	"net/http"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestTransformCSSBasic(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

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
		Output:  ns.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	all := ns.readFile("/public/all.css")
	c.Contains(all, "/f/")
	c.NotContains(all, "/r/")
}

func TestTransformCSSError(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, nil)
	defer ns.clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/all.css": stringHandler{
				contType: cssType,
				body:     `body { derp`,
			},
		}),
		Entries: []string{"/all.css"},
		Output:  ns.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.NotNil(err)
	c.Contains(err, "/all.css")
}
