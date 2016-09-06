package main

import (
	"fmt"
	"os"

	"github.com/thatguystone/cog/cfs"
)

func (ss *siteState) copyFile(src, dst string) {
	err := cfs.Copy(src, dst)
	if err != nil {
		ss.errs.add(src, fmt.Errorf("failed to copy: %v", err))
		return
	}

	info, err := os.Stat(src)
	if err != nil {
		ss.errs.add(src, err)
		return
	}

	os.Chtimes(dst, info.ModTime(), info.ModTime())
	ss.markUsed(dst)
}
