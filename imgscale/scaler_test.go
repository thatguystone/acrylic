package imgscale

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/cog/check"
)

func maybeSkipImageTest(t *testing.T) {
	_, err := exec.LookPath("gm")
	if err != nil {
		t.Skip("gm not installed")
	}
}

func TestImageScalerBasic(t *testing.T) {
	maybeSkipImageTest(t)

	c := check.New(t)

	tmp := internal.NewTmpDir(c, map[string]string{
		"img.gif": string(internal.GifBin),
	})
	defer tmp.Remove()

	h := New(
		Root(tmp.Path(".")),
		Cache(tmp.Path(".cache")))

	tests := []string{
		"",
		"W=10",
		"W=10&H=10",
		"W=10&H=10&Q=90",
		"W=10&H=10&Crop=1&Gravity=nw",
		"&Ext=png",
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", "/img.gif?"+test, nil)
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)
		c.Equal(rr.Code, http.StatusOK, rr.Body.String())
	}
}

// func TestImageScalerError(t *testing.T) {
// 	maybeSkipImageTest(t)

// 	c := check.New(t)

// 	tmp := internal.NewTmpDir(c, map[string]string{
// 		"img.gif": string(internal.GifBin),
// 	})
// 	defer tmp.Remove()

// 	h := NewImageScaler(ImageScalerConfig{
// 		Root: tmp.Path("."),
// 	})

// 	tests := []string{
// 		"&no=100",
// 		"&Ext=does-not-exist",
// 	}

// 	for _, test := range tests {
// 		req := httptest.NewRequest("GET", "/img.gif?"+test, nil)
// 		rr := httptest.NewRecorder()

// 		h.ServeHTTP(rr, req)
// 		c.Equal(rr.Code, http.StatusOK, rr.Body.String())
// 	}
// }
