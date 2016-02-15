package afs

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestMain(m *testing.M) {
	check.Main(m)
}

func TestCopy(t *testing.T) {
	c := check.New(t)

	c.FS.SWriteFile("src", "contents")

	for i := 0; i < 5; i++ {
		err := Copy(c.FS.Path("src"), c.FS.Path("dest"))
		c.MustNotError(err)

		c.FS.SContentsEqual("dest", "contents")
	}
}

func TestDropFirst(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		in  string
		out string
	}{
		{
			in:  "some/long/path",
			out: "long/path",
		},
		{
			in:  "idontuseseps",
			out: "",
		},
		{
			in:  "",
			out: "",
		},
	}

	for _, t := range tests {
		c.Equal(DropFirst(t.in), t.out)
	}
}

func TestDropRoot(t *testing.T) {
	c := check.New(t)

	in := "/some/root/test/another"
	out := "test/another"

	c.Equal(out, DropRoot("/some", "root", in))
	c.Equal(out, DropRoot("/some", "root/", in))
	c.Equal(in, DropRoot("/some", "not/root/", in))
}
