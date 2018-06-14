package testutil

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thatguystone/cog/check"
)

// A TmpDir is used for testing
type TmpDir struct {
	c    *check.C
	root string
}

// NewTmpDir creates a new temp directory
func NewTmpDir(c *check.C, files map[string]string) *TmpDir {
	root, err := ioutil.TempDir("", "acrylic-test-")
	c.Must.Nil(err)

	tmp := TmpDir{
		c:    c,
		root: root,
	}

	defer func() {
		if err != nil {
			tmp.Remove()
		}
	}()

	for path, content := range files {
		path = tmp.Path(path)

		err = os.MkdirAll(filepath.Dir(path), 0750)
		c.Must.Nil(err)

		err = ioutil.WriteFile(path, []byte(content), 0640)
		c.Must.Nil(err)
	}

	return &tmp
}

// Remove removes the temp dir and everything in it
func (tmp *TmpDir) Remove() {
	err := os.RemoveAll(tmp.root)
	tmp.c.Nil(err)
}

// Path gets the path to a file in the temp dir
func (tmp *TmpDir) Path(p string) string {
	return filepath.Join(tmp.root, filepath.Clean(p))
}

func (tmp *TmpDir) walk(path string, cb func(rel, abs string)) {
	filepath.Walk(tmp.Path(path),
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

// DumpTree dumps the FS tree of the temp dir to the test's logger
func (tmp *TmpDir) DumpTree() {
	tmp.c.Helper()
	tmp.c.Logf("Tree rooted at: %q", tmp.root)

	tmp.walk("/", func(rel, abs string) {
		tmp.c.Logf("\t%s", rel)
	})
}

// GetFiles gets a map of all files in the temp dir with their contents
func (tmp *TmpDir) GetFiles() map[string]string {
	m := make(map[string]string)

	tmp.walk("/", func(rel, abs string) {
		b, err := ioutil.ReadFile(abs)
		tmp.c.Must.Nil(err)

		m[rel] = string(b)
	})

	return m
}

// ReadFile reads a file from the temp dir
func (tmp *TmpDir) ReadFile(path string) string {
	b, err := ioutil.ReadFile(tmp.Path(path))
	tmp.c.Must.Nil(err)
	return string(b)
}

// WriteFile writes a file to the temp dir, creating parents as necessary
func (tmp *TmpDir) WriteFile(path string, b string) {
	path = tmp.Path(path)

	err := os.MkdirAll(filepath.Dir(path), 0750)
	tmp.c.Must.Nil(err)

	err = ioutil.WriteFile(path, []byte(b), 0640)
	tmp.c.Must.Nil(err)
}
