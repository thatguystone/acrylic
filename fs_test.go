package toner

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
