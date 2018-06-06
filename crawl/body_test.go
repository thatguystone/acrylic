package crawl

import (
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"testing"

	"github.com/thatguystone/cog/check"
)

const gifType = "image/gif"

var gifBin = []byte{
	0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
	0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
	0x00, 0x3b,
}

func TestServeFile(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, map[string][]byte{
		"/img.gif": gifBin,
		"/all.css": []byte(`body {}`),
	})
	defer ns.clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/img.gif": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, ns.path("/img.gif"))
				}),
			"/all.css": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, ns.path("/all.css"))
				}),
		}),
		Entries: []string{
			"/img.gif",
			"/all.css",
		},
		Output: ns.path("/public"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	ns.checkFileExists("/public/img.gif")
	ns.checkFileExists("/public/all.css")
}

func TestServeFileFingerprint(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, map[string][]byte{
		"/img.gif": gifBin,
	})
	defer ns.clean()

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<img src="img.gif">`,
			},
			"/img.gif": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, ns.path("img.gif"))
				}),
		}),
		Entries: []string{"/"},
		Output:  ns.path("/public"),
		Fingerprint: func(c *Content) bool {
			return filepath.Ext(c.URL.Path) == ".gif"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	ns.dumpTree()

	img := cont.GetPage("/img.gif")
	c.NotLen(img.Fingerprint, 0)
	c.Contains(img.URL.Path, img.Fingerprint)

	ns.checkFileExists("/img.gif")
	ns.checkFileExists("/public/" + path.Base(img.URL.Path))

	index := ns.readFile("/public/index.html")
	c.Contains(index, img.URL.Path)
}

func TestServeFileDefaultContentType(t *testing.T) {
	c := check.New(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept", pathContentType+",*/*")
	resp := httptest.NewRecorder()

	ServeFile(resp, req, "this.is-not-an-ext")

	c.Equal(resp.Code, http.StatusOK)
	c.Contains(resp.Body.String(), DefaultType)
}

func TestServeFileNoAccepts(t *testing.T) {
	c := check.New(t)

	ns := newTestNS(c, map[string][]byte{
		"/img.gif": gifBin,
	})
	defer ns.clean()

	req := httptest.NewRequest("GET", "/stuff", nil)
	resp := httptest.NewRecorder()

	ServeFile(resp, req, ns.path("/img.gif"))

	c.Equal(resp.Code, http.StatusOK)
	c.Equal(resp.Body.Bytes(), gifBin)
}

func TestBodyErrors(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		body    string
		headers http.Header
	}{
		{
			body: `1234`,
			headers: http.Header{
				"Content-Type": {pathContentType},
			},
		},
		{
			body: `{"ContentType":"invalid; ===="}`,
			headers: http.Header{
				"Content-Type": {pathContentType},
			},
		},
	}

	for _, test := range tests {
		resp := httptest.NewRecorder()
		resp.WriteString(test.body)
		resp.HeaderMap = test.headers

		_, err := newBody(resp)
		c.NotNil(err)
	}
}
