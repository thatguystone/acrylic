package toner

import "testing"

func testP2Exec(t *testing.T, in string, out string, files ...testFile) {
	files = append(files, testFile{
		p:  "content/tpl/render.html",
		sc: in,
	})
	tt := testNew(t, true, files)
	defer tt.cleanup()

	tt.checkFile("public/tpl/render.html",
		out)
}

func TestP2Img(t *testing.T) {
	t.Parallel()

	testP2Exec(t,
		`{% img "path.gif" %}`,
		`<img src=../static/img/tpl/path.gif style=width:1px;height:1px;>`,
		testFile{
			p:  "content/tpl/path.gif",
			bc: gifBin,
		})

	testP2Exec(t,
		`{% img "path.gif" width=200 height=100 crop="left" %}`,
		`<img src=../static/img/tpl/path.200x100.cl.gif style=width:200px;height:100px;>`,
		testFile{
			p:  "content/tpl/path.gif",
			bc: gifBin,
		})
}
