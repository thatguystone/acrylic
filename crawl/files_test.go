package crawl

import (
	"testing"

	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/cog/check"
)

const (
	filesSrc     = "files.go"
	filesTestSrc = "files_test.go"
)

func TestParentDirs(t *testing.T) {
	c := check.New(t)

	tests := []struct {
		in  string
		out []string
	}{
		{
			in: "/some/long/path",
			out: []string{
				"/some/long",
				"/some",
			},
		},
		{
			in: "some/long/path",
			out: []string{
				"some/long",
				"some",
			},
		},
		{
			in:  ".",
			out: []string{},
		},
	}

	for _, test := range tests {
		c.Equal(parentDirs(test.in), test.out)
	}
}

func TestFileEquals(t *testing.T) {
	c := check.New(t)

	tmp := internal.NewTmpDir(c, map[string]string{
		"0.txt": "abc",
		"1.txt": "def",
	})
	defer tmp.Remove()

	equal, err := fileEquals(tmp.Path("0.txt"), []byte("abc"))
	c.Nil(err)
	c.True(equal)

	equal, err = fileEquals(tmp.Path("1.txt"), []byte("abc"))
	c.Nil(err)
	c.False(equal)

	equal, err = fileEquals("this does not exist", []byte("abc"))
	c.Nil(err)
	c.False(equal)
}
