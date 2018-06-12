package crawl

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
)

// Config configures what the Crawler does
type Config struct {
	Handler     http.Handler           // Handler to crawl
	Entries     []*url.URL             // Entry points to crawl
	Output      string                 // Build directory
	Transforms  map[string][]Transform // Transforms to apply, by Content-Type
	Fingerprint FingerprintCb          // Fingerprint the page?
	Links       LinkConfig             // What should be done with links?
}

// Crawl performs a crawl with the given config
func Crawl(cfg Config) (Site, error) {
	cr := newCrawler(cfg)

	for _, entry := range cr.cfg.Entries {
		cr.get(entry)
	}

	cr.wg.Wait()

	if len(cr.err) > 0 {
		return Site{}, cr.err
	}

	err := cr.cleanup()
	if err != nil {
		return Site{}, err
	}

	return cr.site, nil
}

type crawler struct {
	cfg        Config
	transforms map[string][]Transform
	wg         sync.WaitGroup

	mtx  sync.Mutex
	err  SiteError
	site Site
	dirs map[string]struct{}
	used map[string]struct{}
}

func newCrawler(cfg Config) *crawler {
	if len(cfg.Entries) == 0 {
		cfg.Entries = []*url.URL{
			{Path: "/"},
		}
	}

	if cfg.Output == "" {
		cfg.Output = "./public"
	}

	if cfg.Fingerprint == nil {
		cfg.Fingerprint = func(*url.URL, string) bool { return false }
	}

	cr := crawler{
		cfg:        cfg,
		transforms: make(map[string][]Transform),
		err:        make(SiteError),
		site: Site{
			urls:  make(map[string]*Page),
			pages: make(map[string]*Page),
			files: make(map[string]*Page),
		},
		dirs: make(map[string]struct{}),
		used: make(map[string]struct{}),
	}

	for contType, ts := range cfg.Transforms {
		cr.addTransforms(contType, ts...)
	}

	// Default transforms always come after user-supplied transforms so that
	// the defaults may work on final, user-provided content.
	for contType, ts := range defaultTransforms {
		cr.addTransforms(contType, ts...)
	}

	return &cr
}

func (cr *crawler) fingerprint(u url.URL, mediaType string) bool {
	return cr.cfg.Fingerprint(&u, mediaType)
}

func (cr *crawler) addTransforms(contType string, ts ...Transform) {
	cr.transforms[contType] = append(cr.transforms[contType], ts...)
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
	cr.mtx.Lock()
	cr.used[file] = struct{}{}
	cr.mtx.Unlock()
}

// Claim the given output file and its parent directories
func (cr *crawler) claimFile(pg *Page, file string) error {
	dirs := make([]string, 0, strings.Count(file, "/"))
	dir := file
	for {
		dir = filepath.Dir(dir)
		if dir == "/" {
			break
		}

		dirs = append(dirs, dir)
	}

	cr.mtx.Lock()
	defer cr.mtx.Unlock()

	using, inUse := cr.site.files[file]
	if inUse {
		return FileAlreadyClaimedError{
			File:  file,
			Owner: using.OrigURL.String(),
		}
	}

	for _, dir := range dirs {
		if _, ok := cr.site.files[dir]; ok {
			return FileDirMismatchError(dir)
		}
	}

	if _, ok := cr.dirs[file]; ok {
		return FileDirMismatchError(file)
	}

	// Don't mark anything until after all checks so that either the claim
	// succeeds entirely or not at all
	cr.site.files[file] = pg
	cr.used[file] = struct{}{}

	for _, dir := range dirs {
		cr.dirs[dir] = struct{}{}
	}

	return nil
}

func (cr *crawler) cleanup() error {
	return nil
}
