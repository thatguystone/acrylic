package acrylic

import (
	"net/url"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestImgNoChange(t *testing.T) {
	c := check.New(t)

	im, err := newImg("some-img.jpg", nil)
	c.Must.Nil(err)
	c.Equal("some-img.jpg", im.resolvedName)
	c.False(im.needsScale())
	c.True(im.isFinalPath)
}

func TestImgNameArgs(t *testing.T) {
	c := check.New(t)

	im, err := newImg("some-img.jpg@h=100&w=100&d=&merp=derp.png", nil)
	c.Must.Nil(err)
	c.Equal("some-img.jpg@w=100&h=100.png", im.resolvedName)
	c.True(im.needsScale())
	c.False(im.isFinalPath)
}

func TestImgOnlyExtChange(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img.jpg@.png", nil)
	c.Must.Nil(err)
	c.Equal("img.jpg@.png", im.resolvedName)
	c.True(im.needsScale())
	c.True(im.isFinalPath)
}

func TestImgArgsBounce(t *testing.T) {
	c := check.New(t)

	paths := []string{
		"",
		`img.jpg`,
		`img@h=10&c=t.jpg`,
		`img@w=10&h=20&c=t&q=95.jpg`,
	}

	for _, path := range paths {
		path := path
		c.Run(path, func(c *check.C) {
			for i := 0; i < 3; i++ {
				im, err := newImg(path, nil)
				c.Must.Nil(err, "round %d", i)

				c.Equal(path, im.resolvedName, "round %d", i)
				path = im.resolvedName
			}
		})
	}
}

func TestImgDefaultArgs(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img.jpg@q=0&d=0", nil)
	c.Must.Nil(err)
	c.Equal(im.quality, 100)
	c.Equal(im.density, 1)
	c.Equal("img.jpg", im.resolvedName)
	c.False(im.needsScale())
	c.False(im.isFinalPath)
}

func TestImgNoDstExt(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img.jpg@h=100", nil)
	c.Must.Nil(err)
	c.Equal(im.h, 100)
	c.Equal("img.jpg@h=100.jpg", im.resolvedName)
	c.True(im.needsScale())
	c.False(im.isFinalPath)
}

func TestImgDstExtArg(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img.jpg@dstExt=test", url.Values{
		"dstExt": {"jpg"},
	})
	c.Must.Nil(err)
	c.Equal("img.jpg", im.resolvedName)
	c.False(im.needsScale())
	c.False(im.isFinalPath)
}

func TestImgDstExtAsNameArg(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img.jpg@dstExt=test", nil)
	c.Must.Nil(err)
	c.Equal("img.jpg@.test", im.resolvedName)
	c.True(im.needsScale())
	c.False(im.isFinalPath)
}

func TestImgPathArgs(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img@w=100&h=100&c=t.jpg", nil)
	c.Must.Nil(err)
	c.Equal(im.w, 100)
	c.Equal(im.h, 100)
	c.Equal("img@w=100&h=100&c=t.jpg", im.resolvedName)
	c.True(im.needsScale())
	c.True(im.isFinalPath)
}

func TestImgArgPrecedence(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img@h=100.jpg", url.Values{
		"h": {"200"},
	})
	c.Must.Nil(err)
	c.Equal(im.h, 200)
}

func TestImgExts(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img.jpg@.gif", nil)
	c.Must.Nil(err)
	c.Equal(im.dstExt, ".gif")

	im, err = newImg("img.jpg", nil)
	c.Must.Nil(err)
	c.Equal(im.dstExt, "")

	im, err = newImg("img.jpg", url.Values{
		"dstExt": {"gif"},
	})
	c.Must.Nil(err)
	c.Equal(im.dstExt, ".gif")
	c.Equal("img.jpg@.gif", im.resolvedName)
}

func TestImgInvalidQueryArgs(t *testing.T) {
	c := check.New(t)

	_, err := newImg("img.jpg", url.Values{
		"h": {"abcd"},
	})
	c.NotNil(err)

	_, err = newImg("img.jpg", url.Values{
		"q": {"abcd"},
	})
	c.NotNil(err)
}

func TestImgInvalidNameArgs(t *testing.T) {
	c := check.New(t)

	_, err := newImg("img@h=abcd.jpg", nil)
	c.NotNil(err)

	_, err = newImg("img@h=-1.jpg", nil)
	c.NotNil(err)
}

func TestImgScaleArgs(t *testing.T) {
	c := check.New(t)

	im, err := newImg("img@h=100&w=100&q=96.jpg", nil)
	c.Must.Nil(err)
	args := imgArgs.cmdArgs(im)
	c.Contains(args, "100x100")
	c.Contains(args, "96")

	im, err = newImg("img@h=100.jpg", nil)
	c.Must.Nil(err)
	args = imgArgs.cmdArgs(im)
	c.Contains(args, "x100")

	im, err = newImg("img@h=100&c=t.jpg", nil)
	c.Must.Nil(err)
	args = imgArgs.cmdArgs(im)
	c.Contains(args, "x100")
	c.Contains(args, "x100^")

	im, err = newImg("img@c=t.jpg", nil)
	c.Must.Nil(err)
	args = imgArgs.cmdArgs(im)
	c.False(im.isFinalPath)
	c.NotContains(args, "-scale")
	c.NotContains(args, "-extent")

	im, err = newImg("img.jpg", nil)
	c.Must.Nil(err)
	args = imgArgs.cmdArgs(im)
	c.Len(args, 0)
}
