package crawl

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/check"
)

type testNS struct {
	c    *check.C
	root string
}

func newTestNS(c *check.C, files map[string][]byte) *testNS {
	root, err := ioutil.TempDir("", "acrylic-test-")
	c.Must.Nil(err)

	ns := testNS{
		c:    c,
		root: root,
	}

	defer func() {
		if err != nil {
			ns.clean()
		}
	}()

	for path, content := range files {
		path = ns.path(path)

		err = os.MkdirAll(filepath.Dir(path), 0750)
		c.Must.Nil(err)

		err = ioutil.WriteFile(path, content, 0600)
		c.Must.Nil(err)
	}

	return &ns
}

func (ns *testNS) clean() {
	err := os.RemoveAll(ns.root)
	ns.c.Nil(err)
}

func (ns *testNS) path(p string) string {
	return filepath.Join(ns.root, filepath.Clean(p))
}

func (ns *testNS) dumpTree() {
	ns.c.Helper()
	ns.c.Logf("Tree rooted at: %q", ns.root)

	filepath.Walk(ns.root, func(path string, info os.FileInfo, err error) error {
		ns.c.Must.Nil(err)

		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(ns.root, path)
		ns.c.Must.Nil(err)

		ns.c.Logf("\t%s", rel)
		return nil
	})
}

func (ns *testNS) checkFileExists(path string) {
	ns.c.Helper()

	ok, err := cfs.FileExists(ns.path(path))
	ns.c.Must.Nil(err)
	ns.c.True(ok, "expected file %q to exist", path)
}

func (ns *testNS) readFile(path string) string {
	ns.c.Helper()

	b, err := ioutil.ReadFile(ns.path(path))
	ns.c.Must.Nil(err)
	return string(b)
}
