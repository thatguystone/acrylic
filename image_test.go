package acrylic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/cog/check"
)

func maybeSkipImageTest(t *testing.T) {
	// t.SkipNow("gm not installed")
}

func TestImageScalerBasic(t *testing.T) {
	maybeSkipImageTest(t)

	c := check.New(t)

	tmp := internal.NewTmpDir(c, map[string]string{
		"img.gif": string(internal.GifBin),
	})
	defer tmp.Remove()

	h := NewImageScaler(ImageScalerConfig{
		Root: tmp.Path("."),
	})

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

func TestImgOptsStrings(t *testing.T) {
	c := check.New(t)

	newInt := func(i int) *int { return &i }
	newStr := func(s string) *string { return &s }

	tests := []struct {
		opts    imgOpts
		query   string
		variant string
	}{
		{
			variant: "img.jpg",
		},
		{
			opts: imgOpts{
				W: newInt(100),
			},
			query:   "?W=100",
			variant: "img-100x.jpg",
		},
		{
			opts: imgOpts{
				H: newInt(100),
			},
			query:   "?H=100",
			variant: "img-x100.jpg",
		},
		{
			opts: imgOpts{
				W: newInt(100),
				H: newInt(100),
			},
			query:   "?H=100&W=100",
			variant: "img-100x100.jpg",
		},
		{
			opts: imgOpts{
				Q: newInt(50),
			},
			query:   "?Q=50",
			variant: "img-q50.jpg",
		},
		{
			opts: imgOpts{
				Crop: true,
			},
			query:   "?Crop=1",
			variant: "img-c.jpg",
		},
		{
			opts: imgOpts{
				Crop:    true,
				Gravity: northWest,
			},
			query:   "?Crop=1&Gravity=nw",
			variant: "img-cnw.jpg",
		},
		{
			opts: imgOpts{
				Ext: newStr(".png"),
			},
			query:   "?Ext=.png",
			variant: "img.png",
		},
	}

	for _, test := range tests {
		c.Equal(test.opts.query(), test.query)
		c.Equal(test.opts.variantName("img.jpg"), test.variant)
	}
}

func TestCropGravities(t *testing.T) {
	c := check.New(t)

	gravities := []cropGravity{
		center,
		northWest,
		north,
		northEast,
		west,
		east,
		southWest,
		south,
		southEast,
	}

	for _, gravity := range gravities {
		c.NotPanics(func() {
			gravity.String()
		})

		c.NotPanics(func() {
			gravity.shortName()
		})

		b, err := gravity.MarshalText()
		c.Nil(err)

		var g cropGravity

		err = g.UnmarshalText(b)
		c.Nil(err)
		c.Equal(gravity, g)
	}
}

func TestCropGravityErrors(t *testing.T) {
	c := check.New(t)

	c.Panics(func() {
		cropGravity(10000).String()
	})

	c.Panics(func() {
		cropGravity(10000).shortName()
	})

	var g cropGravity
	err := g.UnmarshalText([]byte(`100000`))
	c.NotNil(err)
}
