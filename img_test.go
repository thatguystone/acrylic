package acrylic

import (
	"net/url"
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
)

func TestImgName(t *testing.T) {
	c := check.New(t)

	fs, cleanup := c.FS()
	defer cleanup()

	fs.WriteFile("img.gif", gifBin)

	img, err := newImg(url.Values{
		"W":    {"100"},
		"H":    {"100"},
		"D":    {"2"},
		"Crop": {"true"},
		"Ext":  {".png"},
	})
	c.Must.Nil(err)

	base, err := img.cacheName(fs.Path("img.gif"))
	c.Must.Nil(err)

	c.Contains(base, "img.200x200c-")
	c.Contains(base, ".png")
}
