package crawl

import (
	"io"
	"net/url"
)

// Resourcer is the common interface for all resource types
type resourcer interface {
	// Give the resource the global state and base URL
	init(state *state, url *url.URL)

	// Get the list of paths that this resource claims
	pathClaims() []string

	// Get the final path that this resource writes to
	path() string

	// The server said the file hasn't changed, but it still might contain
	// resources that need to be claimed so that they're not deleted. Take
	// care of it.
	recheck(r io.Reader)

	// This is a new resource. Process it and return some content for writing.
	// If nil is returned, no file is created (and any existing file is left
	// untouched).
	process(resp *response) io.Reader
}

type resourceBase struct {
	state *state
	url   *url.URL // URL to which all links are relative
}

func (rsrc *resourceBase) init(state *state, url *url.URL) {
	rsrc.state = state
	rsrc.url = url
}

func (rsrc *resourceBase) pathClaims() []string {
	return []string{
		rsrc.url.Path,
	}
}

func (rsrc *resourceBase) path() string {
	return rsrc.url.Path
}

func (rsrc resourceBase) loadRelative(sURL string) *content {
	url, err := rsrc.url.Parse(sURL)
	if err != nil {
		rsrc.state.Errorf("[rel url] invalid URL %s: %v", sURL, err)
		return nil
	}

	c := rsrc.state.load(url.String())
	return c.follow()
}
