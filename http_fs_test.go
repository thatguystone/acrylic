package acrylic

import (
	"net/http"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestMultiFSBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	fs.SWriteFile("/one/one", "")
	fs.SWriteFile("/two/two", "")
	mfs := MultiFS{
		http.Dir(fs.Path("/one")),
		http.Dir(fs.Path("/two")),
	}

	f, err := mfs.Open("one")
	c.Must.Nil(err)
	f.Close()

	f, err = mfs.Open("two")
	c.Must.Nil(err)
	f.Close()

	_, err = mfs.Open("three")
	c.NotNil(err)
}
