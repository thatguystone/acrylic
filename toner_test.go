package toner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

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
	gifBin = []byte("GIF89a��€��ÿÿÿ���,�������D�;")
	pngBin = []byte("‰PNG  ��� IHDR����������:~›U��� IDATWcø��ZMoñ����IEND®B`‚")

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
			sc: "<body>Blog layout: {{ Page.Content }}\n</body>" +
				"{% js \"layout.js\" %}\n" +
				"{% css \"layout.css\" %}\n" +
				"{% js_all %}\n" +
				"{% css_all %}\n" +
				"{% js \"layout2.js\" %}\n" +
				"{% css \"layout2.css\" %}\n",
		},
		testFile{
			p:  "layouts/blog/layout.js",
			sc: "layout js!",
		},
		testFile{
			p:  "layouts/blog/layout.css",
			sc: "layout css!",
		},
		testFile{
			p:  "layouts/blog/layout2.js",
			sc: "layout 2 js!",
		},
		testFile{
			p:  "layouts/blog/layout2.css",
			sc: "layout 2 css!",
		},
		testFile{
			dir: true,
			p:   "content/blog/empty",
		},
	}
)

func testNew(t *testing.T, build bool, files []testFile) *testToner {
	cfg := Config{
		Root:       filepath.Join("test_data", assert.GetTestName()),
		MinifyHTML: true,
	}

	tt := &testToner{
		Toner: New(cfg),
		a:     assert.From(t),
	}

	tt.createFiles(files)

	if build {
		_, errs := tt.Build()
		tt.a.MustEqual(0, len(errs), "failed to build site; errs=%v", errs)
	}

	return tt
}

func (tt *testToner) createFiles(files []testFile) {
	for _, file := range files {
		p := filepath.Join(tt.cfg.Root, file.p)

		if file.dir {
			err := os.MkdirAll(p, 0700)
			tt.a.MustNotError(err, "failed to create dir %s", p)
		} else {
			f, err := fCreate(p, createFlags, 0600)
			tt.a.MustNotError(err, "failed to create file %s", p)

			if len(file.sc) > 0 {
				_, err = f.Write([]byte(file.sc))
			} else {
				_, err = f.Write(file.bc)
			}

			tt.a.MustNotError(err, "failed to write file %s", p)

			err = f.Close()
			tt.a.MustNotError(err, "failed to write file %s", p)
		}
	}
}

func (tt *testToner) exists(path string) {
	p := filepath.Join(tt.cfg.Root, path)
	_, err := os.Stat(p)
	tt.a.True(err == nil, "file %s does not exist", p)
}

func (tt *testToner) checkFile(path, contents string) {
	f, err := os.Open(filepath.Join(tt.cfg.Root, path))
	tt.a.MustNotError(err, "failed to open %s", path)
	defer f.Close()

	fc, err := ioutil.ReadAll(f)
	tt.a.MustNotError(err, "failed to read %s", path)

	tt.a.Equal(contents, string(fc), "content mismatch for %s", path)
}

func (tt *testToner) checkBinFile(path string, contents []byte) {

}

func (tt *testToner) cleanup() {
	os.RemoveAll(tt.cfg.Root)
}

func TestEmptySite(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, nil)
	defer tt.cleanup()
}

func TestBasicSite(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, basicSite)
	defer tt.cleanup()

	tt.exists("public/blog/post1.js")
	tt.exists("public/blog/post1.css")
	tt.exists("public/blog/post2.js")
	tt.exists("public/blog/post2.css")

	tt.checkFile("public/blog/post1.html",
		`Blog layout:<h1>post 1</h1><p>post 1<script src=post1.js></script><link rel=stylesheet href=post1.css><script src=../layout/blog/layout.js></script><link rel=stylesheet href=../layout/blog/layout.css><script src=../layout/blog/layout2.js></script><link rel=stylesheet href=../layout/blog/layout2.css>`)
	tt.checkFile("public/blog/post2.html",
		`Blog layout:<h1>post 2</h1><p>post 2<script src=post2.js></script><link rel=stylesheet href=post2.css><script src=../layout/blog/layout.js></script><link rel=stylesheet href=../layout/blog/layout.css><script src=../layout/blog/layout2.js></script><link rel=stylesheet href=../layout/blog/layout2.css>`)
}
