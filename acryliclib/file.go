package acryliclib

import (
	"path/filepath"
	"strings"
)

type file struct {
	srcPath    string
	dstPath    string
	isImplicit bool // File represents an implicit _multi or _index
	layoutName string
}

func (f *file) isIndex() bool {
	return strings.HasPrefix(f.srcPath, "index.")
}

func (f *file) isMeta() bool {
	return filepath.Ext(f.srcPath) == ".meta"
}