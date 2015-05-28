package toner

import (
	"os"
	"path/filepath"

	"github.com/rainycape/vfs"
)

func fcreate(
	fs vfs.VFS,
	path string,
	flag int,
	perm os.FileMode) (vfs.WFile, error) {

	dir, _ := filepath.Split(path)

	err := vfs.MkdirAll(fs, dir, 0750)
	if err != nil {
		return nil, err
	}

	return fs.OpenFile(path, flag, perm)
}
