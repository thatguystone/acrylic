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

	tests := []struct {
		root, path string
		out        string
	}{
		{
			root: "root/path/",
			path: "root/path//some/file/",
			out:  "some/file",
		},
		{
			root: "root/path",
			path: "some/file",
			out:  "some/file",
		},
		{
			root: "",
			path: "some/file",
			out:  "some/file",
		},
	}

	for _, test := range tests {
		c.Equal(DropRoot(test.root, test.path), test.out)
	}
}
