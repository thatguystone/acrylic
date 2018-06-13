// Package imgscale implements an on-demand image scaler
package imgscale

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/thatguystone/acrylic/crawl"
	"github.com/thatguystone/acrylic/internal"
)

type scaler struct {
	root  string // Root path for images
	cache string // Where to cache scaled images
	sema  chan struct{}
}

// New creates a handler that scales images on-demand
func New(opts ...Option) http.Handler {
	s := &scaler{
		root:  ".",
		cache: ".cache",
		sema:  make(chan struct{}, runtime.NumCPU()),
	}

	for _, opt := range opts {
		opt.applyTo(s)
	}

	return s
}

func (s *scaler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var args args

	err := args.parse(r.URL.Query())
	if err != nil {
		internal.HTTPError(w, err.Error(), http.StatusBadRequest)
		return
	}

	origPath := filepath.Join(s.root, r.URL.Path)
	origInfo, err := os.Stat(origPath)
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}

		internal.HTTPError(w, err.Error(), status)
		return
	}

	cachePath := filepath.Join(s.cache, r.URL.Path) + args.query()
	cacheInfo, err := os.Stat(cachePath)
	if err != nil && !os.IsNotExist(err) {
		internal.HTTPError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache's ModTime is always synced with orig's, so if unchanged, then cache
	// is still good
	if cacheInfo == nil || !origInfo.ModTime().Equal(cacheInfo.ModTime()) {
		err := s.scale(args, origPath, cachePath, origInfo.ModTime())
		if err != nil {
			internal.HTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	crawl.Variant(w, args.variantName(r.URL.Path))
	crawl.ServeFile(w, r, cachePath)
}

func (s *scaler) scale(args args, src, dst string, origMod time.Time) (err error) {
	err = os.MkdirAll(filepath.Dir(dst), 0777)
	if err != nil {
		return
	}

	// Use a temp file to implement atomic cache writes; specifically, write to
	// the temp file first, then if everything is good, replace any existing
	// cache file with the temp file (on Linux, at least, this is atomic).
	tmpF, err := ioutil.TempFile(filepath.Dir(dst), args.getTempFilePattern(dst))
	if err != nil {
		return err
	}

	tmpPath := tmpF.Name()
	tmpF.Close()

	s.sema <- struct{}{}
	defer func() {
		<-s.sema

		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	err = args.scale(src, tmpPath)
	if err != nil {
		return
	}

	err = os.Chtimes(tmpPath, time.Now(), origMod)
	if err != nil {
		return
	}

	err = os.Rename(tmpPath, dst)
	return
}
