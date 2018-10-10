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
	"github.com/thatguystone/cog/cfs"
)

type imgscale struct {
	root  string // Root path for images
	cache string // Where to cache scaled images
	sema  chan struct{}
}

// New creates a handler that scales images on-demand
func New(opts ...Option) http.Handler {
	isc := &imgscale{
		root:  ".",
		cache: ".cache",
		sema:  make(chan struct{}, runtime.NumCPU()),
	}

	for _, opt := range opts {
		opt.applyTo(isc)
	}

	return isc
}

func (isc *imgscale) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srcPath := filepath.Join(isc.root, r.URL.Path)
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}

		internal.HTTPError(w, err.Error(), status)
		return
	}

	args, err := newArgs(r.URL)
	if err != nil {
		internal.HTTPError(w, err.Error(), http.StatusBadRequest)
		return
	}

	variantName := cfs.DropExt(filepath.Base(srcPath)) + args.nameSuffix()

	cachePath := filepath.Join(isc.cache, r.URL.Path, variantName)
	cacheInfo, err := os.Stat(cachePath)
	if err != nil && !os.IsNotExist(err) {
		internal.HTTPError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache's ModTime is always synced with src's, so if unchanged, then cache
	// is still good
	if cacheInfo == nil || !srcInfo.ModTime().Equal(cacheInfo.ModTime()) {
		err := isc.scale(args, srcPath, cachePath, srcInfo.ModTime())
		if err != nil {
			internal.HTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	crawl.Variant(w, variantName)
	crawl.ServeFile(w, r, cachePath)
}

func (isc *imgscale) scale(
	args args, srcPath, dstPath string, srcMod time.Time) (err error) {

	err = os.MkdirAll(filepath.Dir(dstPath), 0777)
	if err != nil {
		return
	}

	// Use a temp file to implement atomic cache writes; specifically, write to
	// the temp file first, then if everything is good, replace any existing
	// cache file with the temp file (on Linux, at least, this is atomic).
	tmpF, err := ioutil.TempFile(
		filepath.Dir(dstPath),
		"acrylic-*"+filepath.Ext(dstPath))
	if err != nil {
		return
	}

	tmpPath := tmpF.Name()
	tmpF.Close()

	isc.sema <- struct{}{}
	defer func() {
		<-isc.sema

		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	err = args.scale(srcPath, tmpPath)
	if err != nil {
		return
	}

	err = os.Chtimes(tmpPath, time.Now(), srcMod)
	if err != nil {
		return
	}

	err = os.Rename(tmpPath, dstPath)
	return
}
