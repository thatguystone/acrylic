package main

import (
	"testing"

	"github.com/thatguystone/assert"
)

func TestFSChangeExt(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	a.Equal("file.ext", fChangeExt("file.test", "ext"))
	a.Equal("file.ext", fChangeExt("file.test", ".ext"))
	a.Equal("file..ext", fChangeExt("file.test", "..ext"))
	a.Equal("file", fChangeExt("file.test", ""))
}

func TestFSDropRoot(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	in := "/some/root/test/another"
	out := "test/another"

	a.Equal(out, fDropRoot("/some", "root", in))
	a.Equal(out, fDropRoot("/some", "root/", in))
	a.Equal(in, fDropRoot("/some", "not/root/", in))
}
