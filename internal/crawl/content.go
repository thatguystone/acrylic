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
	"time"

	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/cfs"
)

type content struct {
	state *state

	loaded   sync.WaitGroup
	url      *url.URL
	typ      contentType
	cacheMod time.Time // When existing file was changed
	lastMod  time.Time
	rsrc     resourcer
}

func newContent(state *state, sURL string) *content {
	c := &content{
		state: state,
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

	c.updateCacheMod()

	resp := c.doRequest()
	if resp == nil {
		return
	}

	defer resp.Body.Close()

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
		// If the content is up-to-date, then nothing to do here
		return

	case http.StatusOK:
		// Proceed as normal

	default:
		c.state.Errorf("[content] "+
			"failed to load %s: status (%d) %s",
			c, resp.StatusCode, resp.Status)
		return
	}

	c.rsrc = c.typ.newResource()
	if c.rsrc == nil {
		return
	}

	c.rsrc.init(c.state, c.url)
	setLoaded()

	if !c.claim(c.rsrc.pathClaims()) {
		return
	}

	r := c.rsrc.process(resp)
	if r == nil {
		return
	}

	path := c.state.outputPath(c.rsrc.path())
	f, err := cfs.Create(path)
	if err == nil {
		defer f.Close()
		_, err = io.Copy(f, r)
	}

	if err == nil {
		err = f.Close()
	}

	if err == nil && !resp.lastMod.IsZero() {
		err = os.Chtimes(path, resp.lastMod, resp.lastMod)
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

	if !c.cacheMod.IsZero() {
		req.Header.Set("If-Modified-Since",
			c.cacheMod.UTC().Format(http.TimeFormat))
	}

	resp, err := c.state.httpClient.Do(req)
	if err != nil {
		c.state.Errorf("[content] failed to load %s: %v", c, err)
		return nil
	}

	wResp := wrapResponse(resp, c.state)

	c.lastMod = wResp.lastMod
	if wResp.typ != contentBlob {
		c.typ = wResp.typ
	}

	return wResp
}

// Brute-force find a path that this resource might be put at, and get its mod
// time. For a site that builds without errors, this should work. For a site
// with conflicting resources, it might pick the wrong file, but at that
// point, it doesn't matter.
func (c *content) updateCacheMod() {
	paths := possibleResourcePaths(c.state, c.url)

	for _, path := range paths {
		info, err := os.Stat(c.state.outputPath(path))
		switch {
		case err == nil && !info.IsDir():
			c.cacheMod = info.ModTime()
			return

		case err != nil && !os.IsNotExist(err):
			c.state.Errorf("[content] failed to stat %s: %v", path, err)
		}
	}

	return
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
