package crawl

import (
	"net/url"
	"sync"
	"time"
)

type cache struct {
	rmtx    sync.RWMutex
	Entries map[string]cacheEntry
}

type cacheEntry struct {
	Path     string
	ModTime  time.Time
	ContType string
}

const cachePath = ".acrylic-cache"

func newCache() *cache {
	return &cache{
		Entries: map[string]cacheEntry{},
	}
}

// Update the cache.
//
// The given filePath should be the path that the resource writes to that does
// not include the output directory.
func (ch *cache) update(filePath string, url *url.URL, resp *response) {
	ch.rmtx.Lock()
	defer ch.rmtx.Unlock()

	ch.Entries[url.String()] = cacheEntry{
		Path:     filePath,
		ModTime:  resp.lastMod,
		ContType: resp.contType,
	}
}

func (ch *cache) get(url string) cacheEntry {
	ch.rmtx.RLock()
	defer ch.rmtx.RUnlock()

	return ch.Entries[url]
}
