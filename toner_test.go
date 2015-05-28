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

func testNew(t *testing.T, build bool, files []testFile) *testToner {
	t.Parallel()

	cfg := Config{
		Root: "/",
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
			f, err := fcreate(tt.fs, file.p, createFlags, 0600)
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
	testNew(t, true, nil)
}

func TestBasicSite(t *testing.T) {
	tt := testNew(t, true, []testFile{
		testFile{
			p:  "/content/blog/post1.md",
			sc: "---\ntitle: test\n---\n# hey there\n{{ \"test\" }}",
		},
		testFile{
			dir: true,
			p:   "/content/blog/empty",
		},
	})

	tt.checkFile("/public/blog/post1.html",
		"<h1>hey there</h1>")
}
