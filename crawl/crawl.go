// Package crawl implements an HTTP crawler for producing static sites from
// http.Handlers
package crawl

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/thatguystone/acrylic/internal"
)

// Crawl performs a crawl with the given config
func Crawl(h http.Handler, opts ...Option) (Site, error) {
	cr := newCrawler(h, opts...)

	for _, entry := range cr.entries {
		cr.get(entry)
	}

	cr.wg.Wait()

	if len(cr.err) > 0 {
		return Site{}, cr.err
	}

	// Only cleanup if the run succeeded. It would suck to flush the cache
	// because of a user mistake.
	err := cr.finish()
	if err != nil {
		return Site{}, err
	}

	return cr.site, nil
}

type crawler struct {
	handler      http.Handler
	entries      []*url.URL
	output       string
	transforms   map[string][]Transform
	fingerprints fingerprints
	cleanDirs    []string
	wg           sync.WaitGroup

	mtx  sync.Mutex
	err  SiteError
	site Site
	used usedFiles
}

// Absolute paths to all used files and directories
type usedFiles map[string]struct{}

func newCrawler(h http.Handler, opts ...Option) *crawler {
	cr := &crawler{
		handler:    h,
		output:     "./public",
		transforms: make(map[string][]Transform),
		fingerprints: fingerprints{
			cacheFile: filepath.Join(internal.DefaultCacheDir, "fingerprints.json.gz"),
		},
		err: make(SiteError),
		site: Site{
			urls:   make(map[string]*Page),
			pages:  make(map[string]*Page),
			claims: make(map[string]*Page),
		},
		used: make(usedFiles),
	}

	for _, opt := range opts {
		opt.applyTo(cr)
	}

	cr.fingerprints.loadCache()

	if len(cr.entries) == 0 {
		cr.entries = []*url.URL{
			{Path: "/"},
		}
	}

	// Default transforms always come after user-supplied transforms so that the
	// defaults may work on final, user-provided content.
	for mediaType, ts := range defaultTransforms {
		cr.addTransforms(mediaType, ts...)
	}

	return cr
}

func (cr *crawler) shouldFingerprint(u url.URL, mediaType string) bool {
	return cr.fingerprints.should(&u, mediaType)
}

func (cr *crawler) addTransforms(mediaType string, ts ...Transform) {
	cr.transforms[mediaType] = append(cr.transforms[mediaType], ts...)
}

func (cr *crawler) addError(u url.URL, err error) {
	cr.mtx.Lock()
	cr.err.add(u.String(), err)
	cr.mtx.Unlock()
}

func (cr *crawler) get(u *url.URL) *Page {
	uu := normURL(u)
	k := uu.String()

	cr.mtx.Lock()

	pg, ok := cr.site.urls[k]
	if !ok {
		pg = newPage(cr, uu)
		cr.site.urls[k] = pg
	}

	cr.mtx.Unlock()

	return pg
}

// Claim the given url.Path such that only 1 Page owns the path
func (cr *crawler) claimPage(pg *Page, path string) (*Page, bool) {
	cr.mtx.Lock()

	claimer, ok := cr.site.pages[path]
	if !ok {
		cr.site.pages[path] = pg
	}

	cr.mtx.Unlock()

	return claimer, !ok
}

// Mark the given file as used so that it doesn't get deleted
func (cr *crawler) setUsed(file string) {
	file = absPath(file)
	parents := parentDirs(file)

	cr.mtx.Lock()

	cr.used[file] = struct{}{}
	for _, parent := range parents {
		cr.used[parent] = struct{}{}
	}

	cr.mtx.Unlock()
}

// Claim the given output file and its parent directories
func (cr *crawler) claimFile(pg *Page, file string) error {
	parents := parentDirs(file)

	cr.mtx.Lock()
	defer cr.mtx.Unlock()

	if claimer, ok := cr.site.claims[file]; ok {
		if claimer != nil {
			return FileAlreadyClaimedError{
				File:     file,
				OwnerURL: claimer.OrigURL.String(),
			}
		}

		return FileDirMismatchError(file)
	}

	for _, parent := range parents {
		if cr.site.claims[parent] != nil {
			return FileDirMismatchError(parent)
		}
	}

	// Don't mark anything until after all checks so that either the claim
	// succeeds entirely or not at all
	cr.used[file] = struct{}{}
	cr.site.claims[file] = pg

	for _, parent := range parents {
		cr.used[parent] = struct{}{}
		cr.site.claims[parent] = nil
	}

	return nil
}

func (cr *crawler) finish() error {
	dirs := []string{absPath(cr.output)}
	for _, dir := range cr.cleanDirs {
		dirs = append(dirs, absPath(dir))
	}

	if cr.fingerprints.cacheEnabled() {
		cr.setUsed(cr.fingerprints.cacheFile)
	}

	for _, dir := range dirs {
		cr.setUsed(dir)
	}

	for _, dir := range dirs {
		err := cr.cleanDir(dir)
		if err != nil {
			return err
		}
	}

	return cr.fingerprints.saveCache(cr.used)
}

func (cr *crawler) cleanDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Don't fail if the root directory doesn't exist
			if dir == path && os.IsNotExist(err) {
				return nil
			}

			return err
		}

		if _, ok := cr.used[path]; ok {
			return nil
		}

		err = os.RemoveAll(path)
		if err != nil {
			return err
		}

		// No need to continue: everything below this was removed
		return filepath.SkipDir
	})
}
