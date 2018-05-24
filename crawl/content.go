package crawl

import (
	"bytes"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thatguystone/cog/cfs"
)

// Content is what lives at a URL
type Content struct {
	Src         url.URL  // Original source location
	Redirect    *Content // What this redirected to
	Dst         string   // Where to write the body
	Fingerprint string   // Hash of content after all transforms
	cr          *Crawler
	loadWg      sync.WaitGroup
	loaded      bool
}

const (
	// DefaultMime is the default content type that servers typically send
	// back when they can't determine a file's type
	DefaultMime = "application/octet-stream"
)

func newContent(cr *Crawler, u url.URL) *Content {
	c := &Content{
		Src: u,
		cr:  cr,
	}
	c.loadWg.Add(1)

	return c
}

func (c *Content) startLoad() {
	c.cr.wg.Add(1)
	go c.load()
}

func (c *Content) setLoaded() {
	if !c.loaded {
		c.loaded = true
		c.loadWg.Done()
	}
}

// Wait for the Content to finish loading.
func (c *Content) waitLoaded() {
	c.loadWg.Wait()
}

func (c *Content) load() {
	defer c.cr.wg.Done()
	defer c.setLoaded()

	if c.IsExternal() {
		// External resource: nothing to do
		return
	}

	err := c.process()
	if err != nil {
		c.cr.addError(c.Src.String(), err)
	}
}

func (c *Content) process() error {
	req, err := http.NewRequest("GET", c.Src.String(), nil)
	if err != nil {
		panic(err)
	}

	rr := httptest.NewRecorder()
	c.cr.cfg.Handler.ServeHTTP(rr, req)

	switch rr.Code {
	case http.StatusNotModified, http.StatusOK:
		body, err := newBody(rr)
		if err != nil {
			return err
		}

		return c.processBody(body)

	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:

		c.Redirect = c.cr.GetRel(c, rr.HeaderMap.Get("Location"))
		return nil

	default:
		return ResponseError{rr}
	}
}

func (c *Content) processBody(body *body) error {
	err := c.applyTransforms(body)
	if err != nil {
		return err
	}

	// It's necessary to fingerprint after transforms so that any sub-
	// resources with changed fingerprints change this resource's fingerprint.
	if c.cr.cfg.Fingerprint(c) {
		err := c.fingerprint(body)
		if err != nil {
			return err
		}
	}

	dst := c.Src.Path

	// If going to a directory, need to add an index.html
	if strings.HasSuffix(dst, "/") {
		dst += "index.html"
	}

	if c.Fingerprint != "" {
		dst = addFingerprint(dst, c.Fingerprint)
	}

	c.Dst = filepath.Join(c.cr.cfg.Output, dst)
	c.setLoaded()
	c.checkMime(body)

	ok, err := c.cr.claim(c, c.Dst)
	if err != nil || !ok {
		return err
	}

	if body.canSymlink() {
		// Need to mark the src as used so that it doesn't get cleaned up,
		// leaving a broken symlink.
		err := c.cr.setUsed(body.symSrc)
		if err != nil {
			return err
		}

		return os.Symlink(body.symSrc, c.Dst)
	}

	if changed, err := c.bodyChanged(body); err != nil || !changed {
		return err
	}

	f, err := cfs.Create(c.Dst)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = body.buff.WriteTo(f)
	if err != nil {
		return err
	}

	return f.Close()
}

// bodyChanged determines if the dst file doesn't need to be written since
// it's already the same as the source. This helps rsync by not changing mod
// times.
func (c *Content) bodyChanged(body *body) (changed bool, err error) {
	f, err := os.Open(c.Dst)
	if err != nil {
		if os.IsNotExist(err) {
			changed = true
			err = nil
		}

		return
	}

	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return
	}

	if info.Size() != int64(body.buff.Len()) {
		changed = true
		return
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	changed = !bytes.Equal(b, body.buff.Bytes())
	return
}

func (c *Content) applyTransforms(body *body) error {
	transforms := c.cr.transforms[body.mediaType]
	if len(transforms) == 0 {
		return nil
	}

	b, err := body.getContent()
	if err != nil {
		return err
	}

	for _, transform := range transforms {
		b, err = transform(c.cr, c, b)
		if err != nil {
			return err
		}
	}

	body.setContent(b)
	return nil
}

func (c *Content) fingerprint(body *body) error {
	r, err := body.getReader()
	if err != nil {
		return err
	}

	defer r.Close()

	c.Fingerprint, err = fingerprint(r)
	return err
}

// checkMime checks that the file type that a static server will respond with
// for the generated file is consistent with the type that was originally sent
// back.
func (c *Content) checkMime(body *body) {
	ext := filepath.Ext(c.Dst)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = DefaultMime
	}

	guess, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		panic(err)
	}

	if guess != body.mediaType {
		c.cr.addError(c.Src.String(), MimeTypeMismatchError{
			C:     c,
			Ext:   ext,
			Guess: guess,
			Got:   body.contType,
		})
	}
}

// GetLinkTo gets a link that references o's final location (following any
// redirects) from c.
func (c *Content) GetLinkTo(o *Content, link string) string {
	o = o.FollowRedirects()

	if o.IsExternal() {
		return o.Src.String()
	}

	relative := false
	switch c.cr.cfg.Links {
	case AbsoluteLinks:
		relative = false
	case RelativeLinks:
		relative = true
	default:
		relative = !path.IsAbs(link)
	}

	if relative {
		return c.getRelLinkTo(o)
	}

	return c.Src.ResolveReference(&o.Src).String()
}

func (c *Content) getRelLinkTo(o *Content) string {
	const up = "../"

	src := path.Clean(c.Src.Path)
	dst := path.Clean(o.Src.Path)

	start := 0
	for i := 0; i < len(src) && i < len(dst); i++ {
		if src[i] != dst[i] {
			break
		}

		if src[i] == '/' {
			start = i + 1
		}
	}

	var b strings.Builder
	dst = dst[start:]
	dirs := strings.Count(src[start:], "/")

	b.Grow((len(up) * dirs) + len(dst))
	for i := 0; i < dirs; i++ {
		b.WriteString(up)
	}

	b.WriteString(dst)

	return b.String()
}

// FollowRedirects follows all redirects to the final Content
func (c *Content) FollowRedirects() *Content {
	curr := c
	curr.waitLoaded()

	for curr.Redirect != nil {
		curr = curr.Redirect
		curr.waitLoaded()
	}

	return curr
}

// IsExternal checks if this content refers to an external URL
func (c *Content) IsExternal() bool {
	return c.Src.Scheme != "" || c.Src.Opaque != "" || c.Src.Host != ""
}
