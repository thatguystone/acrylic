package main

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestFSChangeExt(t *testing.T) {
	c := check.New(t)

	c.Equal("file.ext", fChangeExt("file.test", "ext"))
	c.Equal("file.ext", fChangeExt("file.test", ".ext"))
	c.Equal("file..ext", fChangeExt("file.test", "..ext"))
	c.Equal("file", fChangeExt("file.test", ""))
}

func TestFSDropRoot(t *testing.T) {
	c := check.New(t)

	in := "/some/root/test/another"
	out := "test/another"

	c.Equal(out, fDropRoot("/some", "root", in))
	c.Equal(out, fDropRoot("/some", "root/", in))
	c.Equal(in, fDropRoot("/some", "not/root/", in))
}
