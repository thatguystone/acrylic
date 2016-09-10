package crawl

import (
	"encoding/json"
	"net/url"
	"os"
	"sync"
	"time"
)

type cache struct {
	Entries map[string]cacheEntry // Current entries

	rmtx       sync.RWMutex
	oldEntries map[string]cacheEntry // Past entries
}

type cacheEntry struct {
	Path     string
	ModTime  time.Time
	ContType string
}

const cachePath = ".acrylic-cache"

func loadCache(cch *cache, path string) error {
	cch.Entries = map[string]cacheEntry{}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}

		return err
	}

	defer f.Close()

	err = json.NewDecoder(f).Decode(cch)
	if err == nil {
		cch.oldEntries = cch.Entries
		cch.Entries = map[string]cacheEntry{}
	}

	return err
}

// Update the cache.
//
// The given filePath should be the path that the resource writes to that does
// not include the output directory.
func (cch *cache) update(filePath string, url *url.URL, resp *response) {
	cch.rmtx.Lock()
	defer cch.rmtx.Unlock()

	cch.Entries[url.String()] = cacheEntry{
		Path:     filePath,
		ModTime:  resp.lastMod,
		ContType: resp.contType,
	}
}

func (cch *cache) get(url string) cacheEntry {
	cch.rmtx.RLock()
	defer cch.rmtx.RUnlock()

	ce, ok := cch.Entries[url]
	if !ok {
		ce = cch.oldEntries[url]
	}

	return ce
}
