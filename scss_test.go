package acrylic

import (
	"strings"
	"testing"

	"github.com/thatguystone/cog/check"
)

type testScss struct {
	scss
	fs      *check.FS
	cleanup func()
}

func newScssTest(t *testing.T) (*check.C, *testScss) {
	c := check.New(t)

	ts := &testScss{}
	ts.fs, ts.cleanup = c.FS()

	return c, ts
}

func (ts *testScss) exit() {
	ts.cleanup()
}

func TestScssBasic(t *testing.T) {
	c, ts := newScssTest(t)
	defer ts.exit()

	ts.fs.SWriteFile("all.scss", `@import "sub"; @import "sub2";`)
	ts.fs.SWriteFile("more/_sub.scss", `.sub {color: #000;}`)
	ts.fs.SWriteFile("more2/_sub2.scss", `.sub2 {color: #fff;}`)

	ts.init(ScssArgs{
		Entry: ts.fs.Path("all.scss"),
		IncludePaths: []string{
			ts.fs.Path("more/"),
			ts.fs.Path("more2/"),
		},
	})

	sheet, _, err := ts.pollChanges()
	c.Must.Nil(err)
	c.LenNot(sheet, 0)
}

func TestScssRecurse(t *testing.T) {
	c, ts := newScssTest(t)
	defer ts.exit()

	ts.fs.SWriteFile("all.scss", ``)
	ts.fs.SWriteFile("more/sub.scss", `.sub {color: #000;}`)
	ts.fs.SWriteFile("more2/sub2.scss", `.sub2 {color: #fff;}`)
	ts.fs.SWriteFile("more2/_mixin.scss", `.mixin {color: #fff;}`)

	ts.init(ScssArgs{
		Entry: ts.fs.Path("all.scss"),
		Recurse: []string{
			ts.fs.Path("/"),
			ts.fs.Path("more/"),
			ts.fs.Path("more2/"),
		},
	})

	sheet, _, err := ts.pollChanges()
	c.Must.Nil(err)
	c.LenNot(sheet, 0)

	css := string(sheet)
	c.Equal(strings.Count(css, ".sub "), 1)
	c.Equal(strings.Count(css, ".sub2 "), 1)
	c.Equal(strings.Count(css, ".mixin "), 0)
}

func TestScssSyntaxError(t *testing.T) {
	c, ts := newScssTest(t)
	defer ts.exit()

	ts.fs.SWriteFile("all.scss", `what is this?`)

	ts.init(ScssArgs{
		Entry: ts.fs.Path("all.scss"),
	})

	_, _, err := ts.pollChanges()
	c.NotNil(err)
}
