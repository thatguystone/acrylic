package crawl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/cog/stringc"
)

// A SiteError is returned when there were any issues crawling the handler
type SiteError map[string][]error

func (err SiteError) add(path string, e error) {
	err[path] = append(err[path], e)
}

func (err SiteError) Error() string {
	var paths []string
	for path := range err {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	var b strings.Builder
	b.WriteString("the following paths have errors:\n")

	for _, path := range paths {
		fmt.Fprintf(&b, internal.Indent+"%q:", path)

		for _, err := range err[path] {
			b.WriteString("\n")
			b.WriteString(stringc.Indent(err.Error(), internal.Indent+internal.Indent))
		}
	}

	return b.String()
}

// A ResponseError describes an error with a single HTTP request
type ResponseError struct {
	Status int
	Body   []byte
}

func (err ResponseError) Error() string {
	var body string
	if len(err.Body) > 0 {
		body = "\n" + stringc.Indent(string(err.Body), internal.Indent)
	}

	return fmt.Sprintf("http error: %d%s", err.Status, body)
}

// A MimeTypeMismatchError indicates that content type for an extension does not
// match the Content-Type that was returned for it.
type MimeTypeMismatchError struct {
	Ext          string // Extension
	Guess        string // Guess from extension
	FromResponse string // What was actually sent
}

func (err MimeTypeMismatchError) Error() string {
	return fmt.Sprintf(
		"extension %q has content type %q, but the response Content-Type was %q",
		err.Ext, err.Guess, err.FromResponse)
}

// A FileAlreadyClaimedError indicates that a page cannot be written because
// another page has already claimed its output path.
type FileAlreadyClaimedError struct {
	File     string // Path to claimed file
	OwnerURL string // What already claimed it
}

func (err FileAlreadyClaimedError) Error() string {
	return fmt.Sprintf(
		"output file %q is already claimed by the page at URL %q",
		err.File, err.OwnerURL)
}

// A FileDirMismatchError indicates that a path wants to be claimed as both a
// file and a directory
type FileDirMismatchError string

func (err FileDirMismatchError) Error() string {
	return fmt.Sprintf(
		"output path %q is used as both a file and a directory",
		string(err))
}

// A RedirectLoopError describes a loop in HTTP redirects
type RedirectLoopError struct {
	Start string
	End   string
}

func (err RedirectLoopError) Error() string {
	return fmt.Sprintf(
		"redirect loop, started at %q, saw %q again",
		err.Start, err.End)
}

// A TooManyRedirectsError indicates that the crawler gave up trying to follow
// redirects because there were too many
type TooManyRedirectsError struct {
	Start string
	End   string
}

func (err TooManyRedirectsError) Error() string {
	return fmt.Sprintf(
		"too many redirects, started at %q, gave up at %q",
		err.Start, err.End)
}
