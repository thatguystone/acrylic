package crawl

import (
	"net/url"
	"path"
	"strings"
)

// Site describes an entire crawled site
type Site struct {
	urls   map[string]*Page // Pages by full URL
	pages  map[string]*Page // Pages by url.Path
	claims map[string]*Page // Pages by absolute path. Dir claim if nil.
}

// Get the Page at the given URL.
func (s *Site) Get(u *url.URL) *Page {
	return s.urls[normURL(u).String()]
}

// GetPage gets the rendered Page at the given url.Path.
func (s *Site) GetPage(page string) *Page {
	return s.pages[cleanURLPath(page)]
}

// GetFile gets the rendered Page that corresponds to a file on disk.
func (s *Site) GetFile(path string) *Page {
	return s.claims[absPath(path)]
}

func normURL(u *url.URL) *url.URL {
	uu := *u

	uu.Path = cleanURLPath(uu.Path)

	// Sort query
	uu.RawQuery = uu.Query().Encode()

	// Has no meaning server-side
	uu.Fragment = ""

	return &uu
}

func cleanURLPath(dirty string) string {
	clean := dirty

	// Some links have bare paths (eg. "google.com"), and cleaning it produces a
	// ".", which is wrong. Also can't set path to "/" since that would turn
	// "google.com" into "google.com/", which isn't what was given.
	if clean != "" {
		clean = path.Clean(clean)

		// path.Clean removes trailing slashes, but they matter here
		if strings.HasSuffix(dirty, "/") && !strings.HasSuffix(clean, "/") {
			clean += "/"
		}
	}

	return clean
}
