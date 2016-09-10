package crawl

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/cfs"
)

type content struct {
	state *state

	loaded sync.WaitGroup
	url    *url.URL
	typ    contentType
	cached cacheEntry
	rsrc   resourcer
}

func newContent(state *state, sURL string) *content {
	c := &content{
		state:  state,
		cached: state.cache.get(sURL),
	}

	var err error
	c.url, err = url.Parse(sURL)
	switch {
	case err != nil:
		// Just use a blank URL: anyone relying on this URL will be OK, and
		// the crawl is going to fail anyway, so no harm done.
		c.url = new(url.URL)
		c.typ = contentExternal

		state.Errorf("[content] invalid URL: %s: %v", sURL, err)

	case c.url.Scheme != "" || c.url.Opaque != "" || c.url.Host != "":
		c.typ = contentExternal
	}

	if c.typ == contentBlob {
		// Set load-pending: need to actually load this thing
		c.loaded.Add(1)

		state.wg.Add(1)
		go c.load()
	}

	return c
}

// Wait for the content to finish loading.
func (c *content) waitLoad() {
	c.loaded.Wait()
}

// Load the content. This is only used for internal content.
func (c *content) load() {
	doned := false
	setLoaded := func() {
		if !doned {
			doned = true
			c.loaded.Done()
		}
	}

	defer c.state.wg.Done()
	defer setLoaded()

	resp := c.doRequest()
	if resp == nil {
		return
	}

	defer resp.Body.Close()

	recheck := false

	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:

		url, err := c.url.Parse(resp.Header.Get("Location"))
		// Any errors should already have been filtered out by net/http itself
		cog.Must(err, "invalid Location header")

		c.url = url
		c.typ = contentRedirect

		return

	case http.StatusNotModified:
		// If the content is up-to-date, it only needs to be rechecked so that
		// any of its dependent resources are claimed
		recheck = true

		resp.contType = c.cached.ContType
		resp.lastMod = c.cached.ModTime

	case http.StatusOK:
		// Proceed as normal

	default:
		c.state.Errorf("[content] "+
			"failed to load %s: status (%d) %s",
			c, resp.StatusCode, resp.Status)
		return
	}

	c.typ = contentTypeFromMime(resp.contType)
	c.rsrc = c.typ.newResource()
	if c.rsrc == nil {
		return
	}

	c.rsrc.init(c.state, c.url)
	setLoaded()

	if !c.claim(c.rsrc.pathClaims()) {
		return
	}

	path := c.rsrc.path()
	c.state.cache.update(path, c.url, resp)

	outPath := c.state.outputPath(path)
	if recheck {
		c.recheck(outPath)
	} else {
		c.process(outPath, resp)
	}
}

func (c *content) recheck(path string) {
	f, err := os.Open(path)
	if err != nil {
		c.state.Errorf("[content] failed to recheck %s: %v",
			c, err)
		return
	}

	defer f.Close()

	c.rsrc.recheck(f)
}

func (c *content) process(path string, resp *response) {
	r := c.rsrc.process(resp)
	if r == nil {
		return
	}

	f, err := cfs.Create(path)
	if err == nil {
		defer f.Close()
		_, err = io.Copy(f, r)
	}

	if err == nil {
		err = f.Close()
	}

	if err != nil {
		c.state.Errorf("[content] failed to process %s: %v",
			c, err)
		return
	}
}

func (c *content) doRequest() *response {
	req, err := http.NewRequest("GET", c.url.String(), nil)
	cog.Must(err, "[content] "+
		"failed to create new request (how did that happen?)")

	if !c.cached.ModTime.IsZero() {
		req.Header.Set("If-Modified-Since",
			c.cached.ModTime.UTC().Format(http.TimeFormat))
	}

	resp, err := c.state.httpClient.Do(req)
	if err != nil {
		c.state.Errorf("[content] failed to load %s: %v", c, err)
		return nil
	}

	return wrapResponse(resp, c.state)
}

// Try to claim the output path for this content's exclusive use.
//
// In the case of two things that have the same path claims but different
// query strings, the first one to claim is the one that will write. The other
// is simply ignored since it's assumed that two things with the same path
// claims are the same thing.
func (c *content) claim(paths []string) bool {
	oc, conflict, ok := c.state.claim(c, paths)
	if ok {
		return true
	}

	oPaths := oc.rsrc.pathClaims()

	fail := len(paths) != len(oPaths)
	if !fail {
		sort.Sort(sort.StringSlice(paths))
		sort.Sort(sort.StringSlice(oPaths))

		for i, path := range paths {
			if filepath.Clean(path) != filepath.Clean(oPaths[i]) {
				fail = true
				break
			}
		}
	}

	if fail {
		c.state.Errorf("[content] "+
			"output conflict: both %s and %s use %s",
			c, oc, conflict)
	}

	return false
}

// Follow all redirects, and gets the final content
func (c *content) follow() *content {
	seen := map[*content]struct{}{}

	// It's possible that this content isn't loaded yet
	c.waitLoad()

	fc := c
	for fc.typ == contentRedirect {
		if _, ok := seen[fc]; ok {
			c.state.Errorf("[content] "+
				"redirect loop detected, starts at %s",
				c)
			return fc
		}

		seen[fc] = struct{}{}

		fc = fc.state.load(fc.url.String())
		fc.waitLoad()
	}

	return fc
}

func (c *content) String() string {
	return fmt.Sprintf("%s (%s)", c.url, c.typ)
}
