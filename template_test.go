package toner

import "testing"

func testTplExec(t *testing.T, in string, out string) {
	tt := testNew(t, true, []testFile{
		testFile{
			p:  "content/tpl/render.html",
			sc: in,
		},
	})

	tt.checkFile("public/tpl/render.html",
		out)
}

func TestTemplateImg(t *testing.T) {
	t.Parallel()

	testTplExec(t,
		`{% img "path.jpg" %}`,
		`<img src=content/tpl/path.jpg>`)
}
