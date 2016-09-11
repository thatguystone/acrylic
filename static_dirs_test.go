package acrylic

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestStaticDirs(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("/one/one", "")
	fs.SWriteFile("/two/two", "")
	sds := staticDirs{
		fs.Path("/one"),
		fs.Path("/two"),
	}

	f, err := sds.Open("one")
	c.Must.Nil(err)
	f.Close()

	f, err = sds.Open("two")
	c.Must.Nil(err)
	f.Close()

	_, err = sds.Open("three")
	c.NotNil(err)
}
