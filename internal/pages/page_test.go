package pages

import (
	"testing"

	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/test"
)

func TestPageRenderErrors(t *testing.T) {
	const path = "blog/post1.html"

	c := test.New(t)

	f := file.New(
		c.FS.Path(path),
		c.FS.Path(""),
		true, c.St)
	c.FS.SWriteFile(path, "page")
	p, err := newP(c.St, f)
	c.MustNotError(err)

	c.Error(p.Render(tmplErrCompiler{}))
	c.Error(p.RenderList(tmplErrCompiler{}, nil))
}
