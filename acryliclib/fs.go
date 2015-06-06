package acryliclib

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const createFlags = os.O_RDWR | os.O_CREATE | os.O_TRUNC

func fCreate(path string, flag int, perm os.FileMode) (*os.File, error) {
	err := fCreateParents(path)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, flag, perm)
}

func fCreateParents(path string) error {
	dir, _ := filepath.Split(path)
	return os.MkdirAll(dir, 0750)
}

func fWrite(path string, c []byte, perm os.FileMode) error {
	dir, _ := filepath.Split(path)
	err := os.MkdirAll(dir, 0750)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, c, perm)
}

func fExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func dExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
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

func fPathCheckFor(in string, any ...string) string {
	for _, a := range any {
		found := strings.HasPrefix(in, filepath.Clean(a)+"/") ||
			strings.HasSuffix(in, filepath.Clean("/"+a)) ||
			strings.Contains(in, filepath.Clean("/"+a)+"/")

		if found {
			return a
		}
	}

	return ""
}

func fDropRoot(root, path string) string {
	if !strings.HasSuffix(root, "/") {
		root += "/"
	}

	if strings.HasPrefix(path, root) {
		return path[len(root):]
	}

	return path
}

func fSrcChanged(src, dst string) bool {
	sstat, serr := os.Stat(src)
	dstat, derr := os.Stat(dst)
	if serr != nil || derr != nil {
		return true
	}

	return !dstat.ModTime().Equal(sstat.ModTime())
}
