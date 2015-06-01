package toner

import (
	"os"
	"path/filepath"
)

type file struct {
	srcPath string
	dstPath string
	info    os.FileInfo
}

func (f *file) isMeta() bool {
	return filepath.Ext(f.srcPath) == ".meta"
}
