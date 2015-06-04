package toner

import (
	"image"
	"sync"
	"testing"

	"github.com/thatguystone/assert"
)

func TestContentGenImageThumbnail(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	type test struct {
		iw, ih int
		sw, sh int
		crop   imgCrop
	}

	tests := []test{
		test{
			iw: 10, ih: 10,
			sw: 5, sh: 5,
			crop: cropLeft,
		},
		test{
			iw: 10, ih: 10,
			sw: 5, sh: 20,
			crop: cropLeft,
		},
		test{
			iw: 10, ih: 10,
			sw: 20, sh: 5,
			crop: cropLeft,
		},
		test{
			iw: 20, ih: 10,
			sw: 20, sh: 5,
			crop: cropLeft,
		},
		test{
			iw: 20, ih: 10,
			sw: 20, sh: 20,
			crop: cropLeft,
		},
		test{
			iw: 10, ih: 20,
			sw: 20, sh: 20,
			crop: cropLeft,
		},
		test{
			iw: 20, ih: 20,
			sw: 20, sh: 20,
			crop: cropLeft,
		},
		test{
			iw: 40, ih: 40,
			sw: 20, sh: 20,
			crop: cropLeft,
		},
		test{
			iw: 20, ih: 20,
			sw: 40, sh: 40,
			crop: cropLeft,
		},
		test{
			iw: 20, ih: 20,
			sw: 30, sh: 40,
			crop: cropLeft,
		},
		test{
			iw: 20, ih: 20,
			sw: 40, sh: 30,
			crop: cropLeft,
		},
	}

	resImgs := make([]image.Image, len(tests))
	wg := sync.WaitGroup{}

	gi := contentGenImg{}
	for i, t := range tests {
		wg.Add(1)
		go func(i int, t test) {
			img := img{
				w:    t.sw,
				h:    t.sh,
				crop: t.crop,
			}

			ig := image.NewGray(image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: t.iw, Y: t.ih},
			})

			resImgs[i] = gi.thumbnailImage(ig, img)

			wg.Done()
		}(i, t)
	}

	wg.Wait()

	for i, igo := range resImgs {
		t := tests[i]
		igob := igo.Bounds()

		a.Equal(
			image.Point{X: t.sw, Y: t.sh},
			image.Point{X: igob.Dx(), Y: igob.Dy()},
			"mismatch at %d: %+v", i, t)
	}
}

func TestContentGenImageResize(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	type test struct {
		iw, ih int
		sw, sh int
		crop   imgCrop
		ow, oh int
	}

	tests := []test{
		test{
			iw: 10, ih: 10,
			sw: 5, sh: 5,
			crop: cropLeft,
			ow:   5, oh: 5,
		},
		test{
			iw: 10, ih: 10,
			sw: 5, sh: 20,
			crop: cropLeft,
			ow:   5, oh: 5,
		},
		test{
			iw: 10, ih: 10,
			sw: 20, sh: 5,
			crop: cropLeft,
			ow:   5, oh: 5,
		},
		test{
			iw: 20, ih: 10,
			sw: 20, sh: 5,
			crop: cropLeft,
			ow:   10, oh: 5,
		},
		test{
			iw: 20, ih: 10,
			sw: 20, sh: 20,
			crop: cropLeft,
			ow:   20, oh: 10,
		},
		test{
			iw: 10, ih: 20,
			sw: 20, sh: 20,
			crop: cropLeft,
			ow:   10, oh: 20,
		},
		test{
			iw: 20, ih: 20,
			sw: 20, sh: 20,
			crop: cropLeft,
			ow:   20, oh: 20,
		},
		test{
			iw: 40, ih: 40,
			sw: 20, sh: 20,
			crop: cropLeft,
			ow:   20, oh: 20,
		},
		test{
			iw: 20, ih: 20,
			sw: 40, sh: 40,
			crop: cropLeft,
			ow:   40, oh: 40,
		},
		test{
			iw: 20, ih: 20,
			sw: 30, sh: 40,
			crop: cropLeft,
			ow:   30, oh: 30,
		},
		test{
			iw: 20, ih: 20,
			sw: 30, sh: 0,
			crop: cropLeft,
			ow:   30, oh: 30,
		},
		test{
			iw: 20, ih: 40,
			sw: 30, sh: 0,
			crop: cropLeft,
			ow:   30, oh: 60,
		},
		test{
			iw: 40, ih: 20,
			sw: 30, sh: 0,
			crop: cropLeft,
			ow:   30, oh: 15,
		},
		test{
			iw: 20, ih: 20,
			sw: 0, sh: 30,
			crop: cropLeft,
			ow:   30, oh: 30,
		},
		test{
			iw: 20, ih: 40,
			sw: 0, sh: 30,
			crop: cropLeft,
			ow:   15, oh: 30,
		},
		test{
			iw: 40, ih: 20,
			sw: 0, sh: 30,
			crop: cropLeft,
			ow:   60, oh: 30,
		},
	}

	resImgs := make([]image.Image, len(tests))
	wg := sync.WaitGroup{}

	gi := contentGenImg{}
	for i, t := range tests {
		wg.Add(1)
		go func(i int, t test) {
			img := img{
				w:    t.sw,
				h:    t.sh,
				crop: t.crop,
			}

			ig := image.NewGray(image.Rectangle{
				Min: image.Point{X: 0, Y: 0},
				Max: image.Point{X: t.iw, Y: t.ih},
			})

			resImgs[i] = gi.resizeImage(ig, img)

			wg.Done()
		}(i, t)
	}

	wg.Wait()

	for i, igo := range resImgs {
		t := tests[i]
		igob := igo.Bounds()

		a.Equal(
			image.Point{X: t.ow, Y: t.oh},
			image.Point{X: igob.Dx(), Y: igob.Dy()},
			"mismatch at %d: %+v", i, t)
	}
}
