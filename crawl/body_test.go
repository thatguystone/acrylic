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

	fs, clean := c.FS()
	defer clean()

	fs.WriteFile("img.gif", gifBin)

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/img.gif": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, fs.Path("img.gif"))
				}),
		}),
		Entries: []string{"/img.gif"},
		Output:  fs.Path("output"),
	}

	_, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	fs.FileExists("img.gif")
	fs.FileExists("output/img.gif")
}

func TestServeFileFingerprint(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.WriteFile("img.gif", gifBin)

	cfg := Config{
		Handler: mux(map[string]http.Handler{
			"/": stringHandler{
				contType: htmlType,
				body:     `<img src="img.gif">`,
			},
			"/img.gif": http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ServeFile(w, r, fs.Path("img.gif"))
				}),
		}),
		Entries: []string{"/"},
		Output:  fs.Path("output"),
		Fingerprint: func(c *Content) bool {
			return filepath.Ext(c.URL.Path) == ".gif"
		},
	}

	cont, err := Crawl(cfg)
	c.Must.Nil(err)
	fs.DumpTree(".")

	img := cont.GetPage("/img.gif")
	c.NotLen(img.Fingerprint, 0)
	c.Contains(img.URL.Path, img.Fingerprint)

	fs.FileExists("img.gif")
	fs.FileExists("output/" + path.Base(img.URL.Path))

	index := fs.SReadFile("output/index.html")
	c.Contains(index, img.URL.Path)
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
