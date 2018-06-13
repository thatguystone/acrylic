package crawl

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thatguystone/cog/check"
)

type tmpDir struct {
	c    *check.C
	root string
}

func newTmpDir(c *check.C, files map[string]string) *tmpDir {
	root, err := ioutil.TempDir("", "acrylic-test-")
	c.Must.Nil(err)

	tmp := tmpDir{
		c:    c,
		root: root,
	}

	defer func() {
		if err != nil {
			tmp.remove()
		}
	}()

	for path, content := range files {
		path = tmp.path(path)

		err = os.MkdirAll(filepath.Dir(path), 0750)
		c.Must.Nil(err)

		err = ioutil.WriteFile(path, []byte(content), 0600)
		c.Must.Nil(err)
	}

	return &tmp
}

func (tmp *tmpDir) remove() {
	err := os.RemoveAll(tmp.root)
	tmp.c.Nil(err)
}

func (tmp *tmpDir) path(p string) string {
	return filepath.Join(tmp.root, filepath.Clean(p))
}

func (tmp *tmpDir) walk(path string, cb func(rel, abs string)) {
	filepath.Walk(tmp.path(path),
		func(path string, info os.FileInfo, err error) error {
			tmp.c.Must.Nil(err)

			if !info.IsDir() {
				rel, err := filepath.Rel(tmp.root, path)
				tmp.c.Must.Nil(err)

				cb("/"+rel, path)
			}

			return nil
		})
}

func (tmp *tmpDir) dumpTree() {
	tmp.c.Helper()
	tmp.c.Logf("Tree rooted at: %q", tmp.root)

	tmp.walk("/", func(rel, abs string) {
		tmp.c.Logf("\t%s", rel)
	})
}

func (tmp *tmpDir) getFiles() map[string]string {
	m := make(map[string]string)

	tmp.walk("/", func(rel, abs string) {
		b, err := ioutil.ReadFile(abs)
		tmp.c.Must.Nil(err)

		m[rel] = string(b)
	})

	return m
}

func (tmp *tmpDir) readFile(path string) string {
	b, err := ioutil.ReadFile(tmp.path(path))
	tmp.c.Must.Nil(err)
	return string(b)
}
