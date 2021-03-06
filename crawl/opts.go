package crawl

import (
	"net/url"
)

// An Option is passed to Crawl() to change default options
type Option interface {
	applyTo(cr *crawler)
}

type option func(cr *crawler)

func (o option) applyTo(cr *crawler) { o(cr) }

// Entry adds entry points to the site
func Entry(u ...*url.URL) Option {
	return option(func(cr *crawler) {
		cr.entries = append(cr.entries, u...)
	})
}

// Output sets where generated files are written
func Output(path string) Option {
	return option(func(cr *crawler) {
		cr.output = path
	})
}

// Transforms appends the given transforms to any existing transforms.
// Transforms are looked up by media type (eg. "text/html", not "text/html;
// charset=utf-8").
func Transforms(transforms map[string][]Transform) Option {
	return option(func(cr *crawler) {
		for mediaType, ts := range transforms {
			cr.addTransforms(mediaType, ts...)
		}
	})
}

// Fingerprint sets the callback that determines if a resource should be
// fingerprinted
func Fingerprint(cb func(u *url.URL, mediaType string) bool) Option {
	return option(func(cr *crawler) {
		cr.fingerprints.cb = cb
	})
}

// FingerprintCache sets the file where the fingerprint cache should be written.
// Set to "" to disable caching.
//
// Note: The file is a gzip-compressed .json file. Name it as you will.
func FingerprintCache(cacheFile string) Option {
	return option(func(cr *crawler) {
		cr.fingerprints.cacheFile = cacheFile
	})
}

// CleanDirs appends the given dirs to the set of dirs that is cleaned after a
// crawl
func CleanDirs(dirs ...string) Option {
	return option(func(cr *crawler) {
		cr.cleanDirs = append(cr.cleanDirs, dirs...)
	})
}
