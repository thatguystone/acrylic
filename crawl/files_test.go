package crawl

import (
	"testing"

	"github.com/thatguystone/cog/check"
)

const (
	filesSrc     = "files.go"
	filesTestSrc = "files_test.go"
)

func TestFileEquals(t *testing.T) {
	c := check.New(t)

	tmp := newTmpDir(c, map[string]string{
		"0.txt": "abc",
		"1.txt": "def",
	})
	defer tmp.remove()

	equal, err := fileEquals(tmp.path("0.txt"), []byte("abc"))
	c.Nil(err)
	c.True(equal)

	equal, err = fileEquals(tmp.path("1.txt"), []byte("abc"))
	c.Nil(err)
	c.False(equal)

	equal, err = fileEquals("this does not exist", []byte("abc"))
	c.Nil(err)
	c.False(equal)
}
