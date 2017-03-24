package acrylic

import "net/http"

// MultiFS builds on top of http.FileSystem, searching each FS for the requested
// file before giving up.
type MultiFS []http.FileSystem

// Open implements http.FileSystem
func (mfs MultiFS) Open(name string) (f http.File, err error) {
	for _, fs := range mfs {
		f, err = fs.Open(name)
		if err == nil {
			return
		}
	}

	return
}
