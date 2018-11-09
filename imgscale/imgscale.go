// Package imgscale implements an on-demand image scaler
package imgscale

import (
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/thatguystone/acrylic/crawl"
	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/acrylic/internal/cache"
	"github.com/thatguystone/cog/cfs"
)

type imgscale struct {
	root  string // Root path for images
	cache *cache.ReadThrough
	sema  chan struct{}
}

// New creates a handler that scales images on-demand
func New(opts ...Option) http.Handler {
	isc := &imgscale{
		root:  ".",
		cache: cache.NewReadThrough(filepath.Join(cache.DefaultDir, "imgs")),
		sema:  make(chan struct{}, runtime.NumCPU()),
	}

	for _, opt := range opts {
		opt.applyTo(isc)
	}

	return isc
}

func (isc *imgscale) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	args, err := newArgs(r.URL)
	if err != nil {
		internal.HTTPError(w, err.Error(), http.StatusBadRequest)
		return
	}

	srcPath := filepath.Join(isc.root, r.URL.Path)

	nameSuffix := args.nameSuffix()
	cacheKeys := []string{nameSuffix}

	cachePath, err := isc.cache.GetPath(srcPath, args.Ext, cacheKeys,
		func(dstPath string) error {
			isc.sema <- struct{}{}
			defer func() { <-isc.sema }()

			return args.scale(srcPath, dstPath)
		})
	switch err.(type) {
	case cache.NoSuchSourceFileError:
		internal.HTTPError(w, err.Error(), http.StatusNotFound)

	default:
		internal.HTTPError(w, err.Error(), http.StatusInternalServerError)

	case nil:
		crawl.Variant(w, cfs.DropExt(filepath.Base(srcPath))+nameSuffix)
		internal.SetMustRevalidate(w)
		crawl.ServeFile(w, r, cachePath)
	}
}
