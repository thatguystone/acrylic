package acrylic

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"sync"

	"github.com/pkg/errors"
)

// Content is what lives at a URL
type Content struct {
	Src         url.URL  // Original source location
	Redirect    *url.URL // Parsed `Location` header, only for redirects
	Fingerprint string   // Body's fingerprint before any transforms
	cr          *Crawl
	loadWg      sync.WaitGroup
	didLoad     bool
}

// Accept is sent in the Accept header when acrylic makes a request for a
// resource.
const Accept = "x-acrylic-path"

func newContent(cr *Crawl, u url.URL) *Content {
	c := &Content{
		Src: u,
		cr:  cr,
	}
	c.loadWg.Add(1)
	return c
}

func (c *Content) setLoaded() {
	if !c.didLoad {
		c.loadWg.Done()
		c.didLoad = true
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
		c.cr.Errorf("[content] failed to process %s: %v", c.Src.String(), err)
	}
}

func (c *Content) process() (err error) {
	rr := c.hit()

	switch rr.Code {
	case http.StatusNotModified, http.StatusOK:
		// Handled below

	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:

		c.Redirect, err = url.Parse(rr.HeaderMap.Get("Location"))
		if err != nil {
			err = errors.Wrap(err, "invalid Location header")
		}
		return

	default:
		return fmt.Errorf("http status=%d", rr.Code)
	}

	resp, err := newResponse(rr)
	if err != nil {
		return
	}

	if c.cr.needsFingerprint(resp.contType, path.Ext(c.Src.Path)) {
		return c.fingerprint(resp)
	}

	return c.processRespBody(resp)
}

func (c *Content) fingerprint(resp response) (err error) {
	var fp string

	body, err := resp.getBody()
	if err == nil {
		fp, err = fingerprint(body)
		body.Close()
	}

	if err != nil {
		return errors.Wrap(err, "failed to fingerprint")
	}

	redir := c.Src
	redir.Path = addFingerprint(addIndex(redir.Path), fp)

	sc, alreadyExists := c.cr.newContent(&redir)
	if alreadyExists {
		return fmt.Errorf("fingerprint: %s already exists", redir.String())
	}

	c.Redirect = &url.URL{
		Path: path.Base(redir.Path),
	}
	c.setLoaded()

	sc.Fingerprint = fp
	sc.processRespBody(resp)

	return
}

func (c *Content) processRespBody(resp response) (err error) {
	c.setLoaded()

	dstPath := c.cr.outputPath(c.Src.Path)
	if transform, ok := transforms[resp.contType]; ok {
		body, err := resp.getBody()
		if err != nil {
			return err
		}

		buff := new(bytes.Buffer)
		err = transform(c, body, buff)
		body.Close()

		if err == nil {
			err = c.cr.save(dstPath, bytes.NewReader(buff.Bytes()))
		}
	} else {
		err = resp.saveTo(c.cr, dstPath)
	}

	err = errors.Wrap(err, "create failed")
	return
}

func (c *Content) hit() (w *httptest.ResponseRecorder) {
	w = httptest.NewRecorder()

	c.cr.Handler.ServeHTTP(w, &http.Request{
		Method: "GET",
		URL:    &c.Src,
		Header: http.Header{
			"Accept": {Accept},
		},
	})
	return
}

func (c *Content) getPathTo(rel string) string {
	u, err := c.Src.Parse(rel)
	if err != nil {
		c.cr.Errorf("[rel url] invalid URL %s: %v", rel, err)
		return ""
	}

	nc := c.cr.getContent(u)
	nc.waitLoaded()
	final, _ := nc.followRedirects(*u, rel)
	return final
}

func (c *Content) followRedirects(u url.URL, rel string) (string, *Content) {
	curr := c
	for curr.Redirect != nil {
		if curr.Redirect.IsAbs() || path.IsAbs(curr.Redirect.Path) || curr.IsExternal() {
			rel = curr.Redirect.String()
			u = *curr.Redirect
		} else {
			rel = path.Join(path.Dir(rel), curr.Redirect.Path)
			u = *u.ResolveReference(curr.Redirect)
		}

		curr = c.cr.getContent(&u)
		curr.waitLoaded()
	}

	return rel, curr
}

// IsExternal checks if this content refers to an external URL
func (c *Content) IsExternal() bool {
	return c.Src.Scheme != "" || c.Src.Opaque != "" || c.Src.Host != ""
}

// FollowRedirects follows all redirects to the final Content
func (c *Content) FollowRedirects() *Content {
	_, curr := c.followRedirects(url.URL{}, "")
	return curr
}
