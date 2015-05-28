package toner

import (
	"os"
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

const createFlags = os.O_RDWR | os.O_CREATE | os.O_TRUNC

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
			f, err := tt.fs.OpenFile(file.p, createFlags, 0600)
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

func TestEmptySite(t *testing.T) {
	testNew(t, true, nil)
}

func TestBasicSite(t *testing.T) {
	testNew(t, true, []testFile{
		testFile{
			dir: true,
			p:   "/content/blog/empty",
		},
		testFile{
			p:  "/content/blog/post1.md",
			sc: "# hey there",
		},
	})
}
