package crawl

import (
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"sync"
)

// Config configures what the Crawler does
type Config struct {
	Handler     http.Handler           // Handler to crawl
	Entries     []string               // Entry points to crawl
	Output      string                 // Build directory
	Transforms  map[string][]Transform // Transforms to apply, by Content-Type
	Fingerprint func(*Content) bool    // Should *Content be fingerprinted?
	Links       LinkConfig             // What should be done with links?
}

type LinkConfig int

const (
	PreserveLinks LinkConfig = iota
	AbsoluteLinks
	RelativeLinks
)

// Crawl performs a crawl with the given config
func Crawl(cfg Config) (map[string]*Content, error) {
	if len(cfg.Entries) == 0 {
		cfg.Entries = []string{"/"}
	}

	if cfg.Fingerprint == nil {
		cfg.Fingerprint = func(*Content) bool { return false }
	}

	cr := Crawler{
		cfg:        cfg,
		transforms: make(map[string][]Transform),
		err:        make(Error),
		content:    make(map[string]*Content),
		pages:      make(map[string]*Content),
		outputs:    make(map[string]*Content),
		used:       make(map[string]struct{}),
	}

	for contType, ts := range cfg.Transforms {
		cr.addTransforms(contType, ts...)
	}

	// Default transforms always come after user-supplied transforms so that
	// the defaults may work on final, user-provided content.
	cr.addTransforms(htmlType, transformHTML)
	cr.addTransforms(cssType, transformCSS)
	cr.addTransforms(jsonType, transformJSON)
	cr.addTransforms(svgType, transformSVG)

	for _, entry := range cr.cfg.Entries {
		cr.Get(entry)
	}

	cr.wg.Wait()

	err := cr.err.getError()
	if err != nil {
		return nil, err
	}

	err = cr.cleanup()
	if err != nil {
		return nil, err
	}

	return cr.content, nil
}

type Crawler struct {
	cfg        Config
	transforms map[string][]Transform
	wg         sync.WaitGroup

	mtx     sync.Mutex
	err     Error
	content map[string]*Content
	pages   map[string]*Content
	outputs map[string]*Content
	used    map[string]struct{}
}

// Get gets Content by raw URL
func (cr *Crawler) Get(rawURL string) *Content {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return cr.GetURL(u)
}

// GetRel gets Content relative to the given Content
func (cr *Crawler) GetRel(c *Content, rel string) *Content {
	relU, err := c.Src.Parse(rel)
	if err != nil {
		panic(err)
	}

	return cr.GetURL(relU)
}

// GetURL gets Content by URL
func (cr *Crawler) GetURL(u *url.URL) *Content {
	uu := *u

	// Avoid "." as a path
	if uu.Path != "" {
		uu.Path = path.Clean(uu.Path)
	}

	// Sort query
	uu.RawQuery = uu.Query().Encode()

	// Has no meaning server-side
	uu.Fragment = ""

	k := uu.String()

	cr.mtx.Lock()

	c, ok := cr.content[k]
	if !ok {
		c = newContent(cr, uu)
		cr.content[k] = c
	}

	cr.mtx.Unlock()

	return c
}

func (cr *Crawler) addTransforms(contType string, ts ...Transform) {
	cr.transforms[contType] = append(cr.transforms[contType], ts...)
}

func (cr *Crawler) cleanup() error {
	return nil
}

func (cr *Crawler) addError(file string, err error) {
	cr.mtx.Lock()
	cr.err.add(file, err)
	cr.mtx.Unlock()
}

func (cr *Crawler) claimPage(c *Content, page string) (*Content, bool) {
	cr.mtx.Lock()

	claimer, ok := cr.pages[page]
	if !ok {
		cr.pages[page] = c
	}

	cr.mtx.Unlock()

	return claimer, !ok
}

func (cr *Crawler) setUsed(file string) error {
	abs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	cr.mtx.Lock()
	cr.used[abs] = struct{}{}
	cr.mtx.Unlock()

	return nil
}

func (cr *Crawler) claimOutput(c *Content, file string) error {
	abs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	cr.mtx.Lock()

	using, inUse := cr.outputs[abs]
	if !inUse {
		cr.outputs[abs] = c
		cr.used[abs] = struct{}{}
	}

	cr.mtx.Unlock()

	if inUse {
		return AlreadyClaimedError{
			Path: abs,
			By:   using,
		}
	}

	return nil
}
