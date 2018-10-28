package crawl

import (
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/thatguystone/cog/cfs"
)

type fingerprints struct {
	cb        func(u *url.URL, mediaType string) bool
	cacheFile string
	rwmtx     sync.RWMutex
	cache     map[string]fingerprintEntry
}

type fingerprintEntry struct {
	H string
	T time.Time
}

func (fps *fingerprints) cacheEnabled() bool {
	return fps.cacheFile != ""
}

// This doesn't return an error: if the file doesn't exist, if anything goes
// wrong, just ignore the error since it's only a cache and can be rebuilt
func (fps *fingerprints) loadCache() {
	if !fps.cacheEnabled() {
		return
	}

	// In case any of the following fails, start out with a blank, writable
	// cache
	fps.cache = make(map[string]fingerprintEntry)

	// Lock while loading, just in case anyone tries to read before the load has
	// finished
	fps.rwmtx.Lock()
	go func() {
		defer fps.rwmtx.Unlock()

		f, err := os.Open(fps.cacheFile)
		if err != nil {
			return
		}

		defer f.Close()

		gz, err := gzip.NewReader(f)
		if err != nil {
			return
		}

		json.NewDecoder(gz).Decode(&fps.cache)
	}()
}

func (fps *fingerprints) saveCache(used usedFiles) error {
	if !fps.cacheEnabled() {
		return nil
	}

	for path := range fps.cache {
		if _, ok := used[path]; !ok {
			delete(fps.cache, path)
		}
	}

	err := filePrepWrite(fps.cacheFile)
	if err != nil {
		return err
	}

	// Careful: return after clearing old cache since the cache might not have
	// been empty before
	if len(fps.cache) == 0 {
		return nil
	}

	f, err := os.Create(fps.cacheFile)
	if err != nil {
		return err
	}

	gz := gzip.NewWriter(f)

	err = json.NewEncoder(gz).Encode(fps.cache)
	if err != nil {
		return err
	}

	err = gz.Close()
	if err != nil {
		return err
	}

	return f.Close()
}

func (fps *fingerprints) should(u *url.URL, mediaType string) bool {
	if fps.cb == nil {
		return false
	}

	return fps.cb(u, mediaType)
}

func (fps *fingerprints) get(resp *response) (string, error) {
	if !fps.cacheEnabled() || !resp.body.canSymlink() {
		return fps.calc(resp)
	}

	absSrc := absPath(resp.body.symSrc)

	srcInfo, err := os.Stat(resp.body.symSrc)
	if err != nil {
		return "", err
	}

	fps.rwmtx.RLock()
	fe, ok := fps.cache[absSrc]
	fps.rwmtx.RUnlock()

	if !ok || !srcInfo.ModTime().Equal(fe.T) {
		h, err := fps.calc(resp)
		if err != nil {
			return "", err
		}

		fe = fingerprintEntry{
			H: h,
			T: srcInfo.ModTime(),
		}

		fps.rwmtx.Lock()
		fps.cache[absSrc] = fe
		fps.rwmtx.Unlock()
	}

	return fe.H, nil
}

func (fps *fingerprints) calc(resp *response) (string, error) {
	r, err := resp.body.reader()
	if err != nil {
		return "", err
	}

	defer r.Close()

	return fingerprint(r)
}

func fingerprint(r io.Reader) (string, error) {
	hash := sha1.New()

	_, err := io.Copy(hash, r)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func addFingerprint(p, fp string) string {
	return cfs.ChangeExt(p, fmt.Sprintf("%s%s", fp, path.Ext(p)))
}
