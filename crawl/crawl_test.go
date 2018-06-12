package crawl

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func TestCrawlClaimCollision(t *testing.T) {
	c := check.New(t)

	gifPrint, err := fingerprint(bytes.NewReader(gifBin))
	c.Must.Nil(err)

	fpPath := "/img." + gifPrint + ".gif"

	tests := []struct {
		paths []string
		err   SiteError
	}{
		{
			paths: []string{
				"/img.gif",
				fpPath,
			},
			err: SiteError{
				fpPath: {
					FileAlreadyClaimedError{
						File:  fpPath,
						Owner: "/img.gif",
					},
				},
			},
		},
		{
			paths: []string{
				fpPath,
				"/img.gif",
			},
			err: SiteError{
				"/img.gif": {
					FileAlreadyClaimedError{
						File:  fpPath,
						Owner: fpPath,
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.paths[0], func(c *check.C) {
			tmp := newTmpDir(c, nil)
			defer tmp.remove()

			cfg := Config{
				Handler: mux(map[string]http.Handler{
					"/img.gif": stringHandler{
						contType: gifType,
						body:     string(gifBin),
					},
					fpPath: stringHandler{
						contType: gifType,
						body:     string(gifBin),
					},
				}),
				Output: tmp.path("/public"),
				Fingerprint: func(u *url.URL, mediaType string) bool {
					return strings.Contains(u.Path, "img.gif")
				},
			}

			cr := newCrawler(cfg)

			for _, path := range test.paths {
				cr.get(&url.URL{Path: path})
				cr.wg.Wait()
			}

			c.Equal(cr.err, test.err)
			c.Equal(cr.err.Error(), test.err.Error())
		})
	}
}

func TestCrawlClaimFileDirMismatch(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		paths []string
		err   SiteError
	}{
		{
			paths: []string{
				"/index",
				"/index/",
			},
			err: SiteError{
				"/index/": {
					FileDirMismatchError("/index"),
				},
			},
		},
		{
			paths: []string{
				"/index/",
				"/index",
			},
			err: SiteError{
				"/index": {
					FileDirMismatchError("/index"),
				},
			},
		},
		{
			paths: []string{
				"/index",
				"/index/nested/page/",
			},
			err: SiteError{
				"/index/nested/page/": {
					FileDirMismatchError("/index"),
				},
			},
		},
		{
			paths: []string{
				"/index/nested/page/",
				"/index",
			},
			err: SiteError{
				"/index": {
					FileDirMismatchError("/index"),
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.paths[0], func(c *check.C) {
			tmp := newTmpDir(c, nil)
			defer tmp.remove()

			cfg := Config{
				Handler: mux(map[string]http.Handler{
					"/index": stringHandler{
						contType: DefaultType,
						body:     `file`,
					},
					"/index/": stringHandler{
						contType: htmlType,
						body:     `dir`,
					},
					"/index/nested/page/": stringHandler{
						contType: htmlType,
						body:     `nested`,
					},
				}),
				Output: tmp.path("/public"),
			}

			cr := newCrawler(cfg)

			for _, path := range test.paths {
				cr.get(&url.URL{Path: path})
				cr.wg.Wait()
			}

			c.Equal(cr.err, test.err)
			c.Equal(cr.err.Error(), test.err.Error())
		})
	}
}
