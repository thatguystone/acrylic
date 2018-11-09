package cache

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/thatguystone/acrylic/internal/testutil"
	"github.com/thatguystone/cog/check"
)

func TestReadThroughBasic(t *testing.T) {
	c := check.New(t)

	tmpDir := testutil.NewTmpDir(c, map[string]string{
		"test.json": "{}",
	})
	defer tmpDir.Remove()

	cache := NewReadThrough(tmpDir.Path("cache"))

	calls := 0
	for i := 0; i < 3; i++ {
		_, err := cache.GetPath(tmpDir.Path("test.json"), "", []string{"a"},
			func(writeTo string) error {
				calls++
				return ioutil.WriteFile(writeTo, []byte("test"), 0600)
			})
		c.Nil(err)
	}

	c.Equal(calls, 1)
}

func TestReadThroughNoSuchSrcPath(t *testing.T) {
	c := check.New(t)

	cache := NewReadThrough("/")

	_, err := cache.GetPath("/does/not/exist", "", nil, func(writeTo string) error {
		return errors.New("should not be called")
	})
	c.Equal(err, NoSuchSourceFileError("/does/not/exist"))
}
