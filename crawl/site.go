package crawl

import (
	"net/url"
	"path"
	"strings"
)

// Site describes an entire crawled site
type Site struct {
	urls  map[string]*Page // Pages by full URL
	pages map[string]*Page // Pages by url.Path
	files map[string]*Page // Pages by absolute output path
}

// Get the Page at the given URL.
func (s *Site) Get(u *url.URL) *Page {
	return s.urls[normURL(u).String()]
}

// GetPage gets the rendered Page at the given url.Path.
func (s *Site) GetPage(page string) *Page {
	return s.pages[cleanPath(page)]
}

// GetFile gets the rendered Page that corresponds to a file in the output
// directory.
//
// For example, the file at "~/site/public/dir/index.html" has the path
// "/dir/index.html".
func (s *Site) GetFile(path string) *Page {
	return s.files[path]
}

func cleanPath(dirty string) string {
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

func normURL(u *url.URL) *url.URL {
	uu := *u

	uu.Path = cleanPath(uu.Path)

	// Sort query
	uu.RawQuery = uu.Query().Encode()

	// Has no meaning server-side
	uu.Fragment = ""

	return &uu
}
