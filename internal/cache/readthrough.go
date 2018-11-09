package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/thatguystone/cog/cfs"
)

// A ReadThrough provides a read-through, disk-based cache for things
// calculated based on some source file's mod time.
type ReadThrough struct {
	root string
}

// NewReadThrough creates a new read-through cache
func NewReadThrough(dir string) *ReadThrough {
	return &ReadThrough{
		root: dir,
	}
}

// Create is called to create a new cache entry. The callback should write data
// to the given path.
type Create func(writeTo string) error

// GetPath gets the cached file from the given srcPath, ext, and keys.
func (c *ReadThrough) GetPath(
	srcPath, ext string, keys []string, create Create) (string, error) {

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", NoSuchSourceFileError(srcPath)
		}

		return "", err
	}

	keys = append([]string{srcPath}, keys...)
	key := calcKey(keys)

	cachePath := filepath.Join(c.root, key[:2], key)
	if ext != "" {
		cachePath = cfs.ChangeExt(cachePath, ext)
	}

	cacheInfo, err := os.Stat(cachePath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// Cache's ModTime is always synced with src's, so if unchanged, then cache
	// is still good
	if cacheInfo == nil || !srcInfo.ModTime().Equal(cacheInfo.ModTime()) {
		err := c.populate(cachePath, srcInfo.ModTime(), create)
		if err != nil {
			return "", err
		}
	}

	return cachePath, nil
}

func (c *ReadThrough) populate(
	dstPath string, srcMod time.Time, create Create) (err error) {

	err = os.MkdirAll(filepath.Dir(dstPath), 0777)
	if err != nil {
		return
	}

	// Use a temp file to implement atomic cache writes; specifically, write to
	// the temp file first, then if everything is good, replace any existing
	// cache file with the temp file (on Linux, at least, this is atomic).
	tmpF, err := ioutil.TempFile(
		filepath.Dir(dstPath),
		"acrylic-*"+filepath.Ext(dstPath))
	if err != nil {
		return
	}

	tmpPath := tmpF.Name()
	tmpF.Close()

	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	err = create(tmpPath)
	if err != nil {
		return
	}

	err = os.Chtimes(tmpPath, time.Now(), srcMod)
	if err != nil {
		return
	}

	return os.Rename(tmpPath, dstPath)
}
