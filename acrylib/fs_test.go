package acrylib

import (
	"testing"

	"github.com/thatguystone/assert"
)

func TestFSDropFirst(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	a.Equal("", fDropFirst(""))
	a.Equal("", fDropFirst("/"))
	a.Equal("", fDropFirst("/t"))
	a.Equal("", fDropFirst("/te"))
	a.Equal("", fDropFirst("/tes"))
	a.Equal("", fDropFirst("/test"))
	a.Equal("", fDropFirst("/test/"))
	a.Equal("it", fDropFirst("/test/it"))
	a.Equal("", fDropFirst("some/"))
	a.Equal("test/path", fDropFirst("some/test/path"))
}

func TestFSChangeExt(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	a.Equal("file.ext", fChangeExt("file.test", "ext"))
	a.Equal("file.ext", fChangeExt("file.test", ".ext"))
	a.Equal("file..ext", fChangeExt("file.test", "..ext"))
	a.Equal("file", fChangeExt("file.test", ""))
}

func TestFSPathCheckFor(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	a.Equal("test", fPathCheckFor("/some/path/test", "test", "or", "something"))
	a.Equal("test", fPathCheckFor("/some/path/test/", "test", "or", "something"))
	a.Equal("test", fPathCheckFor("/some/path/test/another", "test", "or", "something"))
	a.Equal("test", fPathCheckFor("test/some/path", "test", "or", "something"))
	a.Equal("", fPathCheckFor("/some/tests/path", "test", "or", "something"))
}

func TestFSDropRoot(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	in := "/some/root/test/another"
	out := "test/another"

	a.Equal(out, fDropRoot("/some/root", in))
	a.Equal(out, fDropRoot("/some/root/", in))
	a.Equal(in, fDropRoot("/some/not/root/", in))
}
