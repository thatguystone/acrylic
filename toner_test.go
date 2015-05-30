package toner

import (
	"io/ioutil"
	"testing"

	"github.com/rainycape/vfs"
	"github.com/thatguystone/assert"
)

type testToner struct {
	*Toner
	a assert.A
}

type testFile struct {
	dir bool
	p   string
	sc  string
	bc  []byte
}

var (
	basicSite = []testFile{
		testFile{
			p: "content/blog/post1.md",
			sc: "---\ntitle: test\n---\n" +
				"# post 1\n" +
				"{{ \"post 1\" }}\n" +
				"{% js \"post1.js\" %}\n" +
				"{% css \"post1.css\" %}\n",
		},
		testFile{
			p:  "content/blog/post1.js",
			sc: "post 1 js stuff!",
		},
		testFile{
			p:  "content/blog/post1.css",
			sc: "post 1 css stuff!",
		},
		testFile{
			p: "content/blog/post2.md",
			sc: "---\ntitle: post2\n---\n" +
				"# post 2\n" +
				"{{ \"post 2\" }}\n" +
				"{% js \"post2.js\" %}\n" +
				"{% css \"post2.css\" %}\n",
		},
		testFile{
			p:  "content/blog/post2.js",
			sc: "post 2 js stuff!",
		},
		testFile{
			p:  "content/blog/post2.css",
			sc: "post 2 css stuff!",
		},
		testFile{
			p: "layouts/blog/_single.html",
			sc: "<body>{{ Page.Content }}\n</body>" +
				"{% js \"layout.js\" %}\n" +
				"{% css \"layout.css\" %}\n" +
				"{% js_tags %}\n" +
				"{% css_tags %}\n" +
				"{% js \"layout2.js\" %}\n" +
				"{% css \"layout2.css\" %}\n",
		},
		testFile{
			dir: true,
			p:   "content/blog/empty",
		},
	}
)

func testNew(t *testing.T, build bool, files []testFile) *testToner {
	cfg := Config{
		MinifyHTML: true,
	}

	tt := &testToner{
		Toner: newToner(cfg, vfs.Memory()),
		a:     assert.From(t),
	}

	tt.createFiles(files)

	if build {
		err := tt.Build()
		tt.a.MustNotError(err, "failed to build site")
	}

	return tt
}

func (tt *testToner) createFiles(files []testFile) {
	for _, file := range files {
		if file.dir {
			err := vfs.MkdirAll(tt.fs, file.p, 0700)
			tt.a.MustNotError(err, "failed to create dir %s", file.p)
		} else {
			f, err := fCreate(tt.fs, file.p, createFlags, 0600)
			tt.a.MustNotError(err, "failed to create file %s", file.p)

			if len(file.sc) > 0 {
				f.Write([]byte(file.sc))
			} else {
				f.Write(file.bc)
			}

			f.Close()
		}
	}
}

func (tt *testToner) exists(path string) {
	_, err := tt.fs.Stat(path)
	tt.a.True(err == nil, "file %s does not exist", path)
}

func (tt *testToner) checkFile(path, contents string) {
	f, err := tt.fs.Open(path)
	tt.a.MustNotError(err, "failed to open %s", path)
	defer f.Close()

	fc, err := ioutil.ReadAll(f)
	tt.a.MustNotError(err, "failed to read %s", path)

	tt.a.Equal(contents, string(fc), "content mismatch for %s", path)
}

func (tt *testToner) checkBinFile(path string, contents []byte) {

}

func TestEmptySite(t *testing.T) {
	t.Parallel()
	testNew(t, true, nil)
}

func TestBasicSite(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, basicSite)

	tt.exists("public/blog/post1.js")
	tt.exists("public/blog/post1.css")
	tt.exists("public/blog/post2.js")
	tt.exists("public/blog/post2.css")

	tt.checkFile("public/blog/post1.html",
		`<h1>post 1</h1><p>post 1<script src=content/blog/post1.js></script><script src=layouts/blog/layout.js></script><script src=layouts/blog/layout2.js></script><link rel=stylesheet href=content/blog/post1.css><link rel=stylesheet href=layouts/blog/layout.css><link rel=stylesheet href=layouts/blog/layout2.css>`)
	tt.checkFile("public/blog/post2.html",
		`<h1>post 2</h1><p>post 2<script src=content/blog/post2.js></script><script src=layouts/blog/layout.js></script><script src=layouts/blog/layout2.js></script><link rel=stylesheet href=content/blog/post2.css><link rel=stylesheet href=layouts/blog/layout.css><link rel=stylesheet href=layouts/blog/layout2.css>`)
}
