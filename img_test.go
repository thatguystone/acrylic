package acrylic

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/thatguystone/cog/check"
)

var (
	gifBin = []byte{
		0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
		0x00, 0x3b,
	}

	pngHeader = []byte("\x89\x50\x4E\x47\x0D\x0A\x1A\x0A")
)

func newImgTest(t *testing.T) (*check.C, *Image, *check.FS, func()) {
	c := check.New(t)

	fs, clean := c.FS()
	fs.WriteFile("src/img.gif", gifBin)

	im := &Image{
		Root:  fs.Path("src/"),
		Cache: fs.Path("cache/"),
	}

	return c, im, fs, clean
}

func hasConvert() bool {
	path, err := exec.LookPath("convert")
	return path != "" && err == nil
}

func (im *Image) hit(c *check.C, path string, acceptPath bool) *httptest.ResponseRecorder {
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)

		if acceptPath {
			r.Header.Set("Accept", Accept)
		}

		im.ServeHTTP(w, r)

		switch w.Code {
		case 302:
			path = w.HeaderMap.Get("Location")

		default:
			return w
		}
	}

	panic(fmt.Errorf("too many redirects for %s", path))
}

func TestImgAcceptPath(t *testing.T) {
	c, im, _, clean := newImgTest(t)
	defer clean()

	w := im.hit(c, "/img.gif", true)
	c.Equal(w.Code, 200)
	c.Contains(w.Body.String(), "cache/img.")
}

func TestImgScale(t *testing.T) {
	if !hasConvert() {
		t.Skip("Missing convert binary")
	}

	c, im, _, clean := newImgTest(t)
	defer clean()

	w := im.hit(c, "/img.gif?W=10&H=10&D=2&Q=50&Ext=.png&Crop=true", false)
	c.Equal(w.Code, 200)
	c.Contains(w.Body.Bytes(), pngHeader)
}

func TestImgScaleCopy(t *testing.T) {
	c, im, _, clean := newImgTest(t)
	defer clean()

	w := im.hit(c, "/img.gif", false)
	c.Equal(w.Code, 200)
	c.Equal(w.Body.Bytes(), gifBin)
}

func TestImgBadScaleArgs(t *testing.T) {
	c, im, _, clean := newImgTest(t)
	defer clean()

	w := im.hit(c, "/img.gif?invalid=11234", false)
	c.Equal(w.Code, 400)
}

func TestImgBgScaleError(t *testing.T) {
	c, im, _, clean := newImgTest(t)
	defer clean()

	cachePath := im.cachePath(httptest.NewRequest("GET", "/img.gif", nil))
	im.bg.do(cachePath, func() error {
		return errors.New("forced bg error")
	})

	w := httptest.NewRecorder()
	im.serveCache(w, httptest.NewRequest("GET", "/img.gif", nil))
	c.Equal(w.Code, 500)
	c.Equal(strings.TrimSpace(w.Body.String()), "forced bg error")
}

func TestImgCacheName(t *testing.T) {
	c, _, fs, clean := newImgTest(t)
	defer clean()

	img, err := newImg(url.Values{
		"W":    {"100"},
		"H":    {"100"},
		"D":    {"2"},
		"Crop": {"true"},
		"Ext":  {".png"},
	})
	c.Must.Nil(err)

	base, err := img.cacheName(fs.Path("src/img.gif"))
	c.Must.Nil(err)

	c.Contains(base, "img.200x200c-")
	c.Contains(base, ".png")
}

func TestImgCacheNameError(t *testing.T) {
	c := check.New(t)

	img, err := newImg(nil)
	c.Must.Nil(err)

	_, err = img.cacheName("/does/not/exist")
	c.True(os.IsNotExist(err))
}

func TestImgScaleSrcError(t *testing.T) {
	if !hasConvert() {
		t.Skip("Missing convert binary")
	}

	c, _, fs, clean := newImgTest(t)
	defer clean()

	img, err := newImg(nil)
	c.Must.Nil(err)

	err = img.scale("/does/not/exist", fs.Path("src/img.gif"))
	c.Must.NotNil(err)
	c.Contains(err.Error(), "convert: exit status")
}

func TestImgScaleDstError(t *testing.T) {
	c := check.New(t)

	img, err := newImg(nil)
	c.Must.Nil(err)

	err = img.scale("/nope", "/this/is/not/allowed")
	c.True(os.IsPermission(err))
}
