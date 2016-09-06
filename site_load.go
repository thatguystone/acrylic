package main

import (
	"os"
	"path/filepath"

	"github.com/thatguystone/cog/cfs"
)

func (ss *siteState) loadContent(file string, info os.FileInfo) {
	if info.IsDir() {
		return
	}

	var err error

	switch filepath.Ext(file) {
	case ".html":
		var pg *page
		pg, err = newPage(ss, file)
		if err == nil {
			ss.addPage(pg)
		}

	case ".jpg", ".gif", ".png", ".svg":

		ss.loadImg(file, info, true)

	case ".meta":
		// Ignore these

	default:
		ss.loadBlob(file, info)
	}

	if err != nil {
		ss.errorf("failed to load file %s: %v", file, error)
	}
}

func (ss *siteState) walk(dir string, cb func(string, os.FileInfo)) {
	dir = filepath.Join(ss.baseDir, dir)

	if exists, _ := cfs.DirExists(dir); !exists {
		return
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ss.errs.add(path, err)
			return nil
		}

		ss.pool.Do(func() {
			cb(path, info)
		})

		return nil
	})
}

func (ss *siteState) loadAssetImages(file string, info os.FileInfo) {
	if !info.IsDir() {
		switch filepath.Ext(file) {
		case ".jpg", ".gif", ".png", ".svg":
			ss.loadImg(file, info, false)
		}
	}
}

func (ss *siteState) loadPublic(file string, info os.FileInfo) {
	ss.addPublic(file)
}
