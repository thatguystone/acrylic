package acrylic

import (
	"net/url"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestImgBasic(t *testing.T) {
	c := check.New(t)

	im, err := newImg("some-img.jpg", nil)
	c.Must.Nil(err)
	c.Equal("some-img.jpg", im.scaledName())
	c.False(im.needsScale())
	c.True(im.isFinalPath)
}

func TestImgArgsBounce(t *testing.T) {
	c := check.New(t)

	paths := []string{
		"",
		`img:srcExt=.+merp.jpg`,
		`img:c=t;srcExt=.+merp.jpg`,
		`img:w=10;h=20;c=t;q=95;srcExt=.+merp.jpg`,
	}

	for _, path := range paths {
		path := path
		c.Run(path, func(c *check.C) {
			for i := 0; i < 3; i++ {
				im, err := newImg(path, nil)
				c.Must.Nil(err, "round %d", i)

				scaled := im.scaledName()
				c.Equal(path, scaled, "round %d", i)

				path = scaled
			}
		})
	}
}

func TestImgPathArgs(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img:h=100;w=100;d=0;q=0;c=1.2.jpg", nil)
	c.Must.Nil(err)
	c.Equal(im.w, 100)
	c.Equal(im.h, 100)
	c.True(im.needsScale())
	c.True(im.isFinalPath)
}

func TestImgArgPrecedence(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img:h=100.jpg", url.Values{
		"h": {"200"},
	})
	c.Must.Nil(err)
	c.Equal(im.h, 100)
}

func TestImgExts(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img:srcExt=gif.jpg", nil)
	c.Must.Nil(err)
	c.Equal(im.srcExt, ".gif")
	c.Equal(im.dstExt, ".jpg")
}

func TestImgInvalidArgs(t *testing.T) {
	c := check.New(t)

	_, err := newImg("img:h=abcd.jpg", nil)
	c.NotNil(err)

	_, err = newImg("img:h=-1.jpg", nil)
	c.NotNil(err)
}

func TestImgScaleArgs(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img:h=100;w=100;q=96.jpg", nil)
	c.Must.Nil(err)
	args := imgArgs.cmdArgs(im)
	c.Contains(args, "100x100")
	c.Contains(args, "96")

	im, err = newImg("img:h=100.jpg", nil)
	c.Must.Nil(err)
	args = imgArgs.cmdArgs(im)
	c.Contains(args, "x100")

	im, err = newImg("img:h=100;c=t.jpg", nil)
	c.Must.Nil(err)
	args = imgArgs.cmdArgs(im)
	c.Contains(args, "x100")
	c.Contains(args, "x100^")

	im, err = newImg("img.jpg", nil)
	c.Must.Nil(err)
	args = imgArgs.cmdArgs(im)
	c.Len(args, 0)
}
