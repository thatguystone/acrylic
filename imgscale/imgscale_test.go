package imgscale

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/thatguystone/acrylic/internal/testutil"
	"github.com/thatguystone/cog/check"
)

func maybeSkipImageTest(t *testing.T) {
	_, err := exec.LookPath("gm")
	if err != nil {
		t.Skip("gm not installed")
	}
}

func TestImgscaleBasic(t *testing.T) {
	maybeSkipImageTest(t)

	c := check.New(t)

	tests := []struct {
		name     string
		query    string
		contType string
	}{
		{
			name:     "Bounce",
			query:    "",
			contType: "image/gif",
		},
		{
			name:     "ScaleWidth",
			query:    "W=10",
			contType: "image/gif",
		},
		{
			name:     "ScaleWidthHeight",
			query:    "W=10&H=10",
			contType: "image/gif",
		},
		{
			name:     "ScaleQuality",
			query:    "W=10&H=10&Q=90",
			contType: "image/gif",
		},
		{
			name:     "ScaleCropGravity",
			query:    "W=10&H=10&Crop=1&Gravity=nw",
			contType: "image/gif",
		},
		{
			name:     "ChangeExt",
			query:    "&Ext=png",
			contType: "image/png",
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.name, func(c *check.C) {
			tmp := testutil.NewTmpDir(c, map[string]string{
				"img.gif": string(testutil.GifBin),
			})
			defer tmp.Remove()

			h := New(
				Root(tmp.Path(".")),
				Cache(tmp.Path(".cache")))

			req := httptest.NewRequest("GET", "/img.gif?"+test.query, nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)
			c.Equal(rr.Code, http.StatusOK, rr.Body.String())
			c.Equal(rr.HeaderMap.Get("Content-Type"), test.contType)
		})
	}
}

func TestImgscaleError(t *testing.T) {
	maybeSkipImageTest(t)

	c := check.New(t)

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "InvalidArg",
			query: "no=100",
		},
	}

	for _, test := range tests {
		test := test

		c.Run(test.name, func(c *check.C) {
			tmp := testutil.NewTmpDir(c, map[string]string{
				"img.gif": string(testutil.GifBin),
			})
			defer tmp.Remove()

			h := New(
				Root(tmp.Path(".")),
				Cache(tmp.Path(".cache")))

			req := httptest.NewRequest("GET", "/img.gif?"+test.query, nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)
			c.NotEqual(rr.Code, http.StatusOK)
		})
	}
}
