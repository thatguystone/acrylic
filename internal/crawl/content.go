package crawl

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/thatguystone/cog/cfs"
)

type content struct {
	state *crawlState

	loaded  sync.WaitGroup
	url     *url.URL
	isIndex bool
	typ     contentType
	lastMod time.Time
}

func newContent(state *crawlState, sURL string) *content {
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

		state.Errorf("[url parse] invalid URL: %s: %v", sURL, err)

	case c.url.Scheme != "" || c.url.Host != "":
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

// Follow all redirects, and get the final content
func (c *content) follow() *content {
	seen := map[*content]struct{}{}

	// It's possible that this content isn't loaded yet
	c.waitLoad()

	fc := c
	for fc.typ == contentRedirect {
		if _, ok := seen[fc]; ok {
			c.state.Errorf("[content] redirect loop detected starting at %s", c)
			return fc
		}

		seen[fc] = struct{}{}

		fc = fc.state.load(fc.url.String())
		fc.waitLoad()
	}

	return fc
}

// Try to claim the output path, along with any joined implied path, for this
// content's exclusive use
func (c *content) claim(impliedPath string) bool {
	return c.state.claim(c, impliedPath)
}

// Save the content
func (c *content) save(content string) {
	c.saveBytes([]byte(content))
}

func (c *content) saveBytes(content []byte) {
	if c.typ == contentExternal {
		panic(fmt.Errorf("cannot save external content (url=%s)", c))
	}

	impliedPath := ""
	if c.isIndex {
		impliedPath = "index.html"
	}

	if !c.claim(impliedPath) {
		return
	}

	path := filepath.Join(c.state.Output, c.url.Path, impliedPath)
	err := cfs.Write(path, content)
	if err != nil {
		c.state.Errorf("[output] failed to save %s: %v", c, err)
	}
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

	resp, err := c.state.httpClient.Get(c.url.String())
	if err != nil {
		c.state.Errorf("[content] failed to load %s: %v", c, err)
		return
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:

		loc := resp.Header.Get("Location")
		url, err := c.url.Parse(loc)

		if err != nil {
			c.state.Errorf("[content] invalid Location header: %v", err)
		} else {
			c.url = url
			c.typ = contentRedirect
		}

		return

	case http.StatusNotModified:
		// Nothing to do. Everything is great.
		return

	case http.StatusOK:
		// Proceed as normal

	default:
		c.state.Errorf("[content] failed to load %s: status (%d) %s",
			c, resp.StatusCode, resp.Status)
		return
	}

	setLoaded()
	c.process(resp)
}

func (c *content) process(resp *http.Response) {
	lastMod := resp.Header.Get("Last-Modified")
	t, err := time.Parse(http.TimeFormat, lastMod)
	if err != nil && lastMod != "" {
		c.state.Logf("W: [content] invalid Last-Modified header from %s: %v",
			c, err)
	} else {
		c.lastMod = t
	}

	contType := resp.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contType)
	if contType != "" && err != nil {
		c.state.Errorf("[content] invalid content type at %s: %v", c, err)
		return
	}

	c.typ = contentTypeFromMime(mediaType)

	switch c.typ {
	case contentHTML:
		c.processHTML(resp)

	case contentCSS:
		c.processCSS(resp)

	case contentJS, contentBlob:
		c.processBlob(resp)
	}
}

func (c *content) String() string {
	return fmt.Sprintf("%s (%s)", c.url, c.typ)
}
