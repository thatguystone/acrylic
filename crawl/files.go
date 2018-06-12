package crawl

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
)

// fileEquals determines if the contents of the regular file at path is the same
// as the given bytes.
func fileEquals(path string, b []byte) (equal bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}

		return
	}

	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return
	}

	// If not a regular file (ie. if symlink, dir, etc), then it's not equal
	if (info.Mode() & os.ModeType) != 0 {
		return
	}

	if info.Size() != int64(len(b)) {
		return
	}

	fb := make([]byte, len(b))
	_, err = io.ReadFull(f, fb)
	if err != nil {
		return
	}

	equal = bytes.Equal(fb, b)
	return
}

func filePrepWrite(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(filepath.Dir(path), 0750)
}

func cleanTree() {

}
