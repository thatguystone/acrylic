package crawl

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// A Page is a single page in a Site
type Page struct {
	OrigURL     url.URL // URL without any changes
	URL         url.URL // Final URL
	Redirect    *Page   // Where this page redirected to
	OutputPath  string  // Absolute path of output file
	Fingerprint string  // Hash of content after all transforms
	cr          *crawler
	pending     bool           // If load is in-progress
	wg          sync.WaitGroup // For waiting for load to finish
}

// UserAgent is the agent sent with every crawler request
const UserAgent = "acrylic/crawler"

func newPage(cr *crawler, u *url.URL) *Page {
	pg := &Page{
		OrigURL: *u,
		URL:     *u,
		cr:      cr,
	}

	pg.pending = !pg.IsExternal()

	if pg.pending {
		pg.cr.wg.Add(1)
		pg.wg.Add(1)
		go pg.load()
	}

	return pg
}

func (pg *Page) setLoaded() {
	if pg.pending {
		pg.pending = false
		pg.wg.Done()
	}
}

func (pg *Page) waitLoaded() {
	// Unfortunately, this can lead to deadlock if 2+ Contents rely on each
	// other and haven't finished loading. It's quite complex to avoid this
	// (you'd need a full dependency graph since you can have long loops like "a
	// -> b -> c -> d -> a"), so rather than try to put something in that only
	// works in a few, limited cases and gives a false sense of security, let's
	// just consistently deadlock if it comes up.
	//
	// All things considered, this should be quite rare.
	pg.wg.Wait()
}

func (pg *Page) addError(err error) {
	pg.cr.addError(pg.OrigURL, err)
}

func (pg *Page) load() {
	defer pg.cr.wg.Done()
	defer pg.setLoaded()

	req := httptest.NewRequest("GET", pg.OrigURL.String(), nil)
	req.Header.Set("Accept", pathContentType+",*/*")
	req.Header.Set("User-Agent", UserAgent)

	rr := httptest.NewRecorder()

	pg.cr.handler.ServeHTTP(rr, req)

	err := pg.handleResp(rr)
	if err != nil {
		pg.addError(err)
	}
}

func (pg *Page) handleResp(rr *httptest.ResponseRecorder) error {
	resp, err := newResponse(rr)
	if err != nil {
		return err
	}

	switch resp.status {
	case http.StatusOK:
		return pg.render(resp)

	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:

		redirURL, err := pg.OrigURL.Parse(resp.header.Get("Location"))
		if err != nil {
			return err
		}

		pg.Redirect = pg.cr.get(redirURL)
		return nil

	default:
		body, _ := resp.body.get()
		return ResponseError{
			Status: resp.status,
			Body:   bytes.TrimSpace(body),
		}
	}
}

func (pg *Page) render(resp *response) error {
	variant := resp.header.Get(variantHeader)
	if variant != "" {
		u, err := pg.URL.Parse(variant)
		if err != nil {
			return err
		}

		pg.URL = *u
	}

	// Need to claim after any variant changes so that variant paths won't
	// collide
	if claimer, ok := pg.cr.claimPage(pg, pg.URL.Path); !ok {
		pg.setAliasOf(claimer)
		return nil
	}

	needsFingerprint := pg.cr.shouldFingerprint(pg.URL, resp.body.mediaType)
	if !needsFingerprint {
		pg.setOutputPath()
	}

	err := pg.applyTransforms(resp)
	if err != nil {
		return err
	}

	// Fingerprint after transforms so that any sub-resources with changed
	// fingerprints change this resource's fingerprint.
	if needsFingerprint {
		err := pg.fingerprint(resp)
		if err != nil {
			return err
		}

		pg.setOutputPath()
	}

	err = checkServeMime(pg.OutputPath, resp.body.mediaType)
	if err != nil {
		// This is just advisory, so no need to fail hard
		pg.addError(err)
	}

	// Need to be sure that output file and all dirs in between are safe for use
	err = pg.cr.claimFile(pg, pg.OutputPath)
	if err != nil {
		return err
	}

	if resp.body.canSymlink() {
		// Need to mark the src as used so that it doesn't get cleaned up,
		// leaving a broken symlink.
		pg.cr.setUsed(resp.body.symSrc)

		err := filePrepWrite(pg.OutputPath)
		if err != nil {
			return err
		}

		return os.Symlink(resp.body.symSrc, pg.OutputPath)
	}

	// If the file hasn't changed, don't write anything: this is mainly for
	// rsync.
	equal, err := fileEquals(pg.OutputPath, resp.body.b)
	if err != nil || equal {
		return err
	}

	err = filePrepWrite(pg.OutputPath)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pg.OutputPath, resp.body.b, 0640)
}

func (pg *Page) setAliasOf(o *Page) {
	o.waitLoaded()
	pg.OutputPath = o.OutputPath
	pg.Fingerprint = o.Fingerprint
}

func (pg *Page) setOutputPath() {
	outPath := pg.URL.Path

	// If going to a directory, need to add an index.html
	if strings.HasSuffix(outPath, "/") {
		outPath += "index.html"
	}

	if pg.Fingerprint != "" {
		outPath = addFingerprint(outPath, pg.Fingerprint)

		// A fingerprint modifies the dest path, so need to reflect that back in
		// the URL so that everything can be rewritten correctly
		pg.URL.Path = outPath
	}

	pg.OutputPath = absPath(filepath.Join(pg.cr.output, outPath))
	pg.setLoaded()
}

func (pg *Page) applyTransforms(resp *response) error {
	transforms := pg.cr.transforms[resp.body.mediaType]
	if len(transforms) == 0 {
		return nil
	}

	b, err := resp.body.get()
	if err != nil {
		return err
	}

	lr := (*linkResolver)(pg)

	for _, transform := range transforms {
		b, err = transform(lr, b)
		if err != nil {
			return err
		}
	}

	resp.body.set(b)
	return nil
}

func (pg *Page) fingerprint(resp *response) error {
	r, err := resp.body.reader()
	if err != nil {
		return err
	}

	defer r.Close()

	pg.Fingerprint, err = fingerprint(r)
	return err
}

// IsExternal checks if this content refers to an external URL
func (pg *Page) IsExternal() bool {
	return pg.URL.Scheme != "" || pg.URL.Opaque != "" || pg.URL.Host != ""
}

const maxRedirects = 25

func (pg *Page) followRedirects() (*Page, error) {
	curr := pg
	curr.waitLoaded()

	seen := make(map[*Page]struct{})
	for curr.Redirect != nil {
		if _, ok := seen[curr]; ok {
			return nil, RedirectLoopError{
				Start: pg.OrigURL.String(),
				End:   curr.OrigURL.String(),
			}
		}

		seen[curr] = struct{}{}
		if len(seen) > maxRedirects {
			return nil, TooManyRedirectsError{
				Start: pg.OrigURL.String(),
				End:   curr.OrigURL.String(),
			}
		}

		curr = curr.Redirect
		curr.waitLoaded()
	}

	return curr, nil
}

// FollowRedirects follows every redirect to the final Page
func (pg *Page) FollowRedirects() *Page {
	// There's no need to do any checks here as in followRedirects(): it
	// shouldn't be possible to access a page externally if there's an error
	// during Crawl().
	for pg.Redirect != nil {
		pg = pg.Redirect
	}

	return pg
}
