package acrylic

import "net/http"

// FileServer implements an http.Handler that adds caching headers that force
// revalidation to all responses.
func FileServer(root http.FileSystem) http.Handler {
	return fileServer{
		h: http.FileServer(root),
	}
}

type fileServer struct {
	h http.Handler
}

func (h fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)
	h.h.ServeHTTP(w, r)
}

func setCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
}

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
