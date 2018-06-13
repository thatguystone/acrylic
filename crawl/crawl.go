package crawl

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
)

// Config configures what the Crawler does
type Config struct {
	Handler     http.Handler           // Handler to crawl
	Entries     []*url.URL             // Entry points to crawl
	Output      string                 // Build directory
	Transforms  map[string][]Transform // Transforms to apply, by media type
	Fingerprint FingerprintCb          // Fingerprint the page?
	CleanDirs   []string               // Any extra directories to clean
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

	err := cr.clean()
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
	used map[string]struct{} // Absolute paths to all used files and directories
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
			urls:   make(map[string]*Page),
			pages:  make(map[string]*Page),
			claims: make(map[string]*Page),
		},
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

func (cr *crawler) clean() error {
	dirs := []string{absPath(cr.cfg.Output)}
	for _, dir := range cr.cfg.CleanDirs {
		dirs = append(dirs, absPath(dir))
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

	return nil
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
