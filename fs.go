package toner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rainycape/vfs"
)

func fCreate(
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

func fDropFirst(path string) string {
	if len(path) == 0 {
		return ""
	}

	idx := strings.IndexRune(path[1:], os.PathSeparator)
	if idx == -1 {
		return ""
	}

	return path[idx+2:]
}

func fChangeExt(path, ext string) string {
	rext := filepath.Ext(path)

	if len(ext) > 0 && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	return path[0:len(path)-len(rext)] + ext
}

func fRelPath(rel, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(rel, path)
}
