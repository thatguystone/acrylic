package blob

import "os"

type blob struct {
	src string
	dst string
}

func (ss *siteState) copyBlobs() {
	for _, b := range ss.blobs {
		func(b *blob) {
			ss.pool.Do(func() {
				ss.copyFile(b.src, b.dst)
			})
		}(b)
	}
}

func (ss *siteState) loadBlob(file string, info os.FileInfo) {
	_, _, _, _, _, dst := ss.getOutInfo(file, ss.cfg.ContentDir, false)
	ss.addBlob(file, dst)
}
