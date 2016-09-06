package imgs

import (
	"fmt"
	"sync"
	"testing"

	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/pool"
	"github.com/thatguystone/acrylic/internal/test"
)

func TestScale(t *testing.T) {
	c := test.New(t)

	tests := []struct {
		w, h    int
		crop    bool
		quality int
	}{
		{},
		{
			w: 100, h: 100,
			crop:    true,
			quality: 99,
		},
	}

	wg := sync.WaitGroup{}

	write := func(i int) *Img {
		name := fmt.Sprintf("%d.gif", i)
		c.FS.WriteFile(name, test.GifBin)

		f := file.New(c.FS.Path(name), c.FS.Path(""), false, c.St)

		img, err := newImg(c.St, f, true)
		c.MustNotError(err)
		c.True(img.IsGif())

		return img
	}

	scale := func(i int) {
		defer wg.Done()
		test := tests[i]

		img := write(i)
		path := img.Scale(test.w, test.h, test.crop, test.quality)

		c.NotEqual(path, "")
	}

	for i := 0; i < 2; i++ {
		wg.Add(len(tests))
		pool.Pool(&c.St.Run, func() {
			for i := range tests {
				go scale(i)
			}
			wg.Wait()
		})

		c.St.Cfg.CacheBust = false
	}

	var img *Img
	c.Equal(img.Scale(0, 0, true, 100), "<IMAGE NOT FOUND>")
}
