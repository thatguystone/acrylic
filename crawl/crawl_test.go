package crawl

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thatguystone/acrylic/internal/testutil"
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

func TestCrawlClean(t *testing.T) {
	c := check.New(t)

	tmp := testutil.NewTmpDir(c, map[string]string{
		"/public/dir/dir/file": ``,
		"/public/dir/file":     ``,
		"/public/file":         ``,
		"/cache0/dir/dir/file": ``,
		"/cache0/dir/file":     ``,
		"/cache0/file":         ``,
		"/cache0/nested/file":  ``,
		"/cache1/file":         ``,
	})
	defer tmp.Remove()

	cr := newCrawler(nil,
		Output(tmp.Path("/public")),
		CleanDirs(
			tmp.Path("/cache0"),
			tmp.Path("/cache0/nested"),
			tmp.Path("/cache1"),
			tmp.Path("/does-not-exist")))
	cr.setUsed(tmp.Path("/public/file"))
	cr.setUsed(tmp.Path("/cache0/file"))
	err := cr.clean()
	c.Nil(err)

	c.Equal(tmp.GetFiles(), map[string]string{
		"/public/file": ``,
		"/cache0/file": ``,
	})
}

func TestCrawlClaimCollision(t *testing.T) {
	c := check.New(t)

	gifPrint, err := fingerprint(bytes.NewReader(testutil.GifBin))
	c.Must.Nil(err)

	fpPath := "/img." + gifPrint + ".gif"

	tests := []struct {
		paths  []string
		getErr func(tmp *testutil.TmpDir) SiteError
	}{
		{
			paths: []string{
				"/img.gif",
				fpPath,
			},
			getErr: func(tmp *testutil.TmpDir) SiteError {
				return SiteError{
					fpPath: {
						FileAlreadyClaimedError{
							File:     tmp.Path(filepath.Join("public", fpPath)),
							OwnerURL: "/img.gif",
						},
					},
				}
			},
		},
		{
			paths: []string{
				fpPath,
				"/img.gif",
			},
			getErr: func(tmp *testutil.TmpDir) SiteError {
				return SiteError{
					"/img.gif": {
						FileAlreadyClaimedError{
							File:     tmp.Path(filepath.Join("public", fpPath)),
							OwnerURL: fpPath,
						},
					},
				}
			},
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.paths[0], func(c *check.C) {
			tmp := testutil.NewTmpDir(c, nil)
			defer tmp.Remove()

			cr := newCrawler(
				mux(map[string]http.Handler{
					"/img.gif": stringHandler{
						contType: testutil.GifType,
						body:     string(testutil.GifBin),
					},
					fpPath: stringHandler{
						contType: testutil.GifType,
						body:     string(testutil.GifBin),
					},
				}),
				Output(tmp.Path("/public")),
				Fingerprint(func(u *url.URL, mediaType string) bool {
					return strings.Contains(u.Path, "img.gif")
				}))

			for _, path := range test.paths {
				cr.get(&url.URL{Path: path})
				cr.wg.Wait()
			}

			err := test.getErr(tmp)

			c.Equal(cr.err, err)
			c.Equal(cr.err.Error(), err.Error())
		})
	}
}

func TestCrawlClaimFileDirMismatch(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		paths  []string
		getErr func(tmp *testutil.TmpDir) SiteError
	}{
		{
			paths: []string{
				"/index",
				"/index/",
			},
			getErr: func(tmp *testutil.TmpDir) SiteError {
				path := tmp.Path(filepath.Join("public", "index"))
				return SiteError{
					"/index/": {
						FileDirMismatchError(path),
					},
				}
			},
		},
		{
			paths: []string{
				"/index/",
				"/index",
			},
			getErr: func(tmp *testutil.TmpDir) SiteError {
				path := tmp.Path(filepath.Join("public", "index"))
				return SiteError{
					"/index": {
						FileDirMismatchError(path),
					},
				}
			},
		},
		{
			paths: []string{
				"/index",
				"/index/nested/page/",
			},
			getErr: func(tmp *testutil.TmpDir) SiteError {
				path := tmp.Path(filepath.Join("public", "index"))
				return SiteError{
					"/index/nested/page/": {
						FileDirMismatchError(path),
					},
				}
			},
		},
		{
			paths: []string{
				"/index/nested/page/",
				"/index",
			},
			getErr: func(tmp *testutil.TmpDir) SiteError {
				path := tmp.Path(filepath.Join("public", "index"))
				return SiteError{
					"/index": {
						FileDirMismatchError(path),
					},
				}
			},
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.paths[0], func(c *check.C) {
			tmp := testutil.NewTmpDir(c, nil)
			defer tmp.Remove()

			cr := newCrawler(
				mux(map[string]http.Handler{
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
				Output(tmp.Path("/public")))

			for _, path := range test.paths {
				cr.get(&url.URL{Path: path})
				cr.wg.Wait()
			}

			err := test.getErr(tmp)

			c.Equal(cr.err, err)
			c.Equal(cr.err.Error(), err.Error())
		})
	}
}
