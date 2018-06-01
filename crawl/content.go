package crawl

import (
	"bytes"
	"fmt"
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
	Src         url.URL  // Original URL
	Redirect    *Content // What this redirected to
	Dst         url.URL  // Final, resolved URL
	OutputPath  string   // File where this is stored
	Fingerprint string   // Hash of content after all transforms
	cr          *Crawler
	load        struct {
		done bool
		wg   sync.WaitGroup
	}
}

const (
	// DefaultType is the default content type that servers typically send back
	// when they can't determine a file's type
	DefaultType = "application/octet-stream"

	// UserAgent is the agent sent with every crawler request
	UserAgent = "acrylic/crawler"
)

func newContent(cr *Crawler, u url.URL) (c *Content) {
	c = &Content{
		Src: u,
		Dst: u,
		cr:  cr,
	}

	if c.IsExternal() {
		c.load.done = true
	} else {
		c.cr.wg.Add(1)
		c.load.wg.Add(1)
		go c.doLoad()
	}

	return
}

func (c *Content) setLoaded() {
	if !c.load.done {
		c.load.done = true
		c.load.wg.Done()
	}
}

func (c *Content) waitLoaded() {
	// Unfortunately, this can lead to deadlock if 2+ Contents rely on each
	// other and haven't finished loading. It's quite complex to avoid this
	// (you'd need a full dependency graph since you can have long loops like "a
	// -> b -> c -> d -> a"), so rather than try to put something in that only
	// works in a few, limited cases and gives a false sense of security, let's
	// just consistently deadlock if it comes up.
	//
	// All things considered, this should be quite rare.
	c.load.wg.Wait()
}

func (c *Content) doLoad() {
	defer c.cr.wg.Done()
	defer c.setLoaded()

	err := c.doRequest()
	if err != nil {
		c.cr.addError(c.Src.String(), err)
	}
}

func (c *Content) doRequest() error {
	req, err := http.NewRequest("GET", c.Src.String(), nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Accept", pathContentType+",*/*")
	req.Header.Set("User-Agent", UserAgent)

	resp := httptest.NewRecorder()
	c.cr.cfg.Handler.ServeHTTP(resp, req)

	switch resp.Code {
	case http.StatusOK:
		return c.render(resp)

	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:

		c.Redirect = c.cr.GetRel(c, resp.HeaderMap.Get("Location"))
		return nil

	default:
		return ResponseError{resp}
	}
}

func (c *Content) render(resp *httptest.ResponseRecorder) error {
	body, err := newBody(resp)
	if err != nil {
		return err
	}

	variant := resp.HeaderMap.Get(variantHeader)
	if variant != "" {
		dst, err := c.Dst.Parse(variant)
		if err != nil {
			return err
		}

		c.Dst = *dst
	}

	if claimer, ok := c.cr.claimPage(c, c.Dst.Path); !ok {
		c.setAliasOf(claimer)
		return nil
	}

	needsFingerprint := c.cr.cfg.Fingerprint(c)
	if !needsFingerprint {
		c.finalizeDest()
	}

	err = c.applyTransforms(body)
	if err != nil {
		return err
	}

	// It's necessary to fingerprint after transforms so that any sub-resources
	// with changed fingerprints change this resource's fingerprint.
	if needsFingerprint {
		err := c.fingerprint(body)
		if err != nil {
			return err
		}

		c.finalizeDest()
	}

	c.checkMime(body)

	// Even though path has already been checked, it's possible that someone is
	// writing paths with hashes that conflict. This shouldn't ever happen, but
	// let's just be sure.
	err = c.cr.claimOutput(c, c.OutputPath)
	if err != nil {
		return err
	}

	if body.canSymlink() {
		// Need to mark the src as used so that it doesn't get cleaned up,
		// leaving a broken symlink.
		c.cr.setUsed(body.symSrc)
		return os.Symlink(body.symSrc, c.OutputPath)
	}

	if changed, err := c.bodyChanged(body); err != nil || !changed {
		return err
	}

	f, err := cfs.Create(c.OutputPath)
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

func (c *Content) setAliasOf(o *Content) {
	o.waitLoaded()
	c.Dst.Path = o.Dst.Path
	c.OutputPath = o.OutputPath
	c.Fingerprint = o.Fingerprint
}

func (c *Content) finalizeDest() {
	// If going to a directory, need to add an index.html
	if strings.HasSuffix(c.Dst.Path, "/") {
		c.Dst.Path += "index.html"
	}

	if c.Fingerprint != "" {
		c.Dst.Path = addFingerprint(c.Dst.Path, c.Fingerprint)
	}

	c.OutputPath = filepath.Join(c.cr.cfg.Output, c.Dst.Path)
	c.setLoaded()
}

// bodyChanged determines if the dst file doesn't need to be written since
// it's already the same as the source. This helps rsync by not changing mod
// times.
func (c *Content) bodyChanged(body *body) (changed bool, err error) {
	f, err := os.Open(c.OutputPath)
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
	ext := filepath.Ext(c.OutputPath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = DefaultType
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

	fmt.Println(o.Src.String(), link)

	if o.IsExternal() {
		return o.Src.String()
	}

	// relative := false
	// switch c.cr.cfg.Links {
	// case AbsoluteLinks:
	// 	relative = false
	// case RelativeLinks:
	// 	relative = true
	// default:
	//	to := url.Parse(link)
	// 	relative = !to.IsAbs()
	// }

	// if relative {
	// 	return c.getRelLinkTo(o)
	// }

	return c.Dst.ResolveReference(&o.Dst).String()
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

	seen := make(map[*Content]struct{})
	for curr.Redirect != nil {
		if _, ok := seen[curr]; ok {
			panic(fmt.Errorf(
				"redirect loop, starting at %q, saw %q again",
				c.Src.String(), curr.Src.String()))
		}

		seen[curr] = struct{}{}
		if len(seen) > 25 {
			panic(fmt.Errorf(
				"too many redirects, starting at %q, gave up at %q",
				c.Src.String(), curr.Src.String()))
		}

		curr = curr.Redirect
		curr.waitLoaded()
	}

	return curr
}

// IsExternal checks if this content refers to an external URL
func (c *Content) IsExternal() bool {
	return c.Src.Scheme != "" || c.Src.Opaque != "" || c.Src.Host != ""
}
