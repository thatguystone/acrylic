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
		`<img src=path.gif>`,
		testFile{
			p:  "content/tpl/path.gif",
			bc: gifBin,
		})
}
