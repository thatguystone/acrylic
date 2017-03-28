package acrylic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestFileServerBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("/file", "")
	h := FileServer(http.Dir(fs.Path("/")))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/file", nil))

	c.Equal(w.Code, 200)
	c.Contains(w.HeaderMap.Get("Cache-Control"), "max-age=0")
	c.Contains(w.HeaderMap.Get("Cache-Control"), "must-revalidate")
}

func TestMultiFSBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("/one/one", "")
	fs.SWriteFile("/two/two", "")
	mfs := MultiFS{
		http.Dir(fs.Path("/one")),
		http.Dir(fs.Path("/two")),
	}

	f, err := mfs.Open("one")
	c.Must.Nil(err)
	f.Close()

	f, err = mfs.Open("two")
	c.Must.Nil(err)
	f.Close()

	_, err = mfs.Open("three")
	c.NotNil(err)
}
