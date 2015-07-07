package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func fCreateParents(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0750)
}

func fCreate(path string) (*os.File, error) {
	err := fCreateParents(path)
	if err != nil {
		return nil, err
	}

	return os.Create(path)
}

func fWrite(path string, c []byte) error {
	err := fCreateParents(path)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, c, 0640)
}

func fCopy(src, dst string) error {
	if !fSrcChanged(src, dst) {
		return nil
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}

	defer s.Close()

	err = fCreateParents(dst)
	if err != nil {
		return err
	}

	d, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer d.Close()

	_, err = io.Copy(d, s)

	return err
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

func fSrcChanged(src, dst string) bool {
	sstat, serr := os.Stat(src)
	dstat, derr := os.Stat(dst)
	if serr != nil || derr != nil {
		return true
	}

	return !dstat.ModTime().Equal(sstat.ModTime())
}

func fDropRoot(base, root, path string) string {
	root = filepath.Join(base, root) + "/"

	if strings.HasPrefix(path, root) {
		return path[len(root):]
	}

	return path
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
