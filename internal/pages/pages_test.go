package pages

import (
	"fmt"
	"testing"

	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/pool"
	"github.com/thatguystone/acrylic/internal/test"
	"github.com/thatguystone/cog/check"
)

type testPs struct {
	*Ps
	c *test.C
}

func TestMain(m *testing.M) {
	check.Main(m)
}

func newTest(t *testing.T) (*test.C, *testPs) {
	c := test.New(t)
	tps := &testPs{
		Ps: New(c.St),
		c:  c,
	}

	return c, tps
}

func (tps *testPs) load(path string) error {
	f := file.New(
		tps.c.FS.Path(path),
		tps.c.FS.Path(""),
		true, tps.c.St)
	return tps.Load(f)
}

func (tps *testPs) mustLoad(path string) {
	tps.c.MustNotError(tps.load(path))
}

func (tps *testPs) render() {
	pool.Pool(&tps.st.Run, func() {
		tps.AllLoaded()
	})

	pool.Pool(&tps.st.Run, func() {
		tps.RenderPages(tmplCompiler{})
	})

	pool.Pool(&tps.st.Run, func() {
		tps.RenderListPages(tmplCompiler{})
	})
}

func TestBasic(t *testing.T) {
	c, tps := newTest(t)

	for i := 0; i < 15; i++ {
		path := fmt.Sprintf("blog/post%d.html", i)
		c.FS.SWriteFile(path, "page")
		tps.mustLoad(path)
	}

	c.FS.SWriteFile("blog/index.html", "---\nlist_page: true\n---\n")
	tps.mustLoad("blog/index.html")

	tps.render()
	c.MustTrue(tps.st.Errs.Ok())

	c.FS.DumpTree("")

	for i := 0; i < 15; i++ {
		path := fmt.Sprintf("public/blog/post%d/index.html", i)
		c.FS.FileExists(path)
	}

	c.FS.FileExists("public/blog/index.html")
	c.FS.FileExists("public/blog/page/2/index.html")
	c.FS.FileExists("public/blog/page/4/index.html")
}

func TestPageFields(t *testing.T) {
	c, tps := newTest(t)

	path := "blog/page1.html"
	content := "content" + moreScissors + "more content"
	c.FS.SWriteFile(path,
		"---\ntitle: page title\n---\n"+
			scissors+content+scissorsEnd)
	tps.mustLoad(path)

	tps.render()
	c.MustTrue(tps.st.Errs.Ok())

	p := tps.byCat["blog"][0]
	c.Equal(p.Title, "page title")
	c.Equal(p.Content, content)
	c.Equal(p.Summary, "content")
}

func TestErrors(t *testing.T) {
	const path = "blog/index.html"

	c, tps := newTest(t)

	c.Error(tps.load(path))

	c.FS.SWriteFile(path, "---\npage")
	c.Error(tps.load(path))

	c.FS.SWriteFile(path, "---\npage\n---\n")
	c.Error(tps.load(path))
}
