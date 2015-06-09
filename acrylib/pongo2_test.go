package acrylib

import "testing"

func testP2Exec(t *testing.T, in string, out string, files ...testFile) {
	tt := testNew(t, true, nil, append(files,
		testFile{
			p:  "content/tpl/render.html",
			sc: in,
		})...)
	defer tt.cleanup()

	tt.contents("public/tpl/render.html",
		out)
}

func TestP2ImgBasic(t *testing.T) {
	t.Parallel()
	testP2Exec(t,
		`{% img "path.gif" %}`,
		`<img src=path.gif style=width:1px;height:1px;>`,
		testFile{
			p:  "content/tpl/path.gif",
			bc: gifBin,
		})
}

func TestP2ImgOptions(t *testing.T) {
	t.Parallel()
	testP2Exec(t,
		`{% img "path.gif" width=20 height=10 crop="left" %}`,
		`<img src=path.20x10.cl.gif style=width:20px;height:10px;>`,
		testFile{
			p:  "content/tpl/path.gif",
			bc: gifBin,
		})
}

func TestP2ContentRel(t *testing.T) {
	t.Parallel()

	tt := testNew(t, true, nil,
		testFile{
			p: "content/test.md",
		},
		testFile{
			p:  "layouts/sub/_single.html",
			sc: "relUrl={% url Page.Meta.parent|content_rel %}",
		},
		testFile{
			p:  "content/sub/index.md",
			sc: "---\nparent: ../test.md\n---\ntest",
		},
	)
	defer tt.cleanup()

	tt.contents("public/sub/index.html",
		"relUrl=../test.html")
}
