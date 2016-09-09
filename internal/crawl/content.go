package crawl

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/thatguystone/cog"
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

// Try to claim the output path for this content's exclusive use
func (c *content) claim() (string, bool) {
	path, impliedPath := c.outputPath()
	return path, c.state.claim(c, impliedPath)
}

func (c *content) save(content string) {
	c.saveBytes([]byte(content))
}

func (c *content) saveBytes(content []byte) {
	c.saveReader(bytes.NewReader(content))
}

func (c *content) saveReader(content io.Reader) {
	if c.typ == contentExternal {
		panic(fmt.Errorf("cannot save external content (url=%s)", c))
	}

	path, ok := c.claim()
	if !ok {
		return
	}

	f, err := cfs.Create(path)
	defer f.Close()

	if err == nil {
		_, err = io.Copy(f, content)
	}

	if err == nil {
		err = f.Close()
	}

	if err == nil && !c.lastMod.IsZero() {
		err = os.Chtimes(path, c.lastMod, c.lastMod)
	}

	if err != nil {
		c.state.Errorf("[output] failed to save %s: %v", c, err)
	}
}

func (c *content) outputPath() (path, impliedPath string) {
	if c.isIndex {
		impliedPath = "index.html"
	}

	path = filepath.Join(c.state.Output, c.url.Path, impliedPath)

	return
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

	outPath, _ := c.outputPath()
	info, err := os.Stat(outPath)
	if err == nil {
		c.lastMod = info.ModTime()
	}

	req, err := http.NewRequest("GET", c.url.String(), nil)
	cog.Must(err, "failed to create new request (how did that happen?)")

	if !c.lastMod.IsZero() {
		req.Header.Set("If-Modified-Since",
			c.lastMod.UTC().Format(http.TimeFormat))
	}

	resp, err := c.state.httpClient.Do(req)
	if err != nil {
		c.state.Errorf("[content] failed to load %s: %v", c, err)
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
		// The content is up-to-date. Just need to claim it so it doesn't get
		// deleted
		c.claim()
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
