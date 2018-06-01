package crawl

import (
	"fmt"
	"net/http/httptest"
	"sort"
	"strings"

	"github.com/thatguystone/cog/stringc"
)

// A Error is returned when there were any issues crawling the handler
type Error map[string][]error

func (err Error) getError() error {
	if len(err) == 0 {
		return nil
	}

	return err
}

func (err Error) add(path string, e error) {
	err[path] = append(err[path], e)
}

const ErrIndent = "    "

func (err Error) Error() string {
	var paths []string
	for path := range err {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	var b strings.Builder
	b.WriteString("the following paths have errors:\n")

	for _, path := range paths {
		fmt.Fprintf(&b, ErrIndent+"%q\n", path)

		for _, err := range err[path] {
			b.WriteString(stringc.Indent(err.Error(), ErrIndent+ErrIndent))
			b.WriteString("\n")
		}
	}

	return b.String()
}

type ResponseError struct {
	*httptest.ResponseRecorder
}

func (err ResponseError) Error() string {
	var body string
	if err.Body.Len() > 0 {
		body = "\n" + stringc.Indent(err.Body.String(), ErrIndent)
	}

	return fmt.Sprintf("http error: %d%s", err.Code, body)
}

type MimeTypeMismatchError struct {
	C     *Content
	Ext   string // Extension
	Guess string
	Got   string
}

func (err MimeTypeMismatchError) Error() string {
	return fmt.Sprintf(
		"extension %q has content type %q, but the response Content-Type was %q",
		err.Ext, err.Guess, err.Got)
}

type AlreadyClaimedError struct {
	Path string   // Path to claimed file
	By   *Content // What already claimed it
}

func (err AlreadyClaimedError) Error() string {
	return fmt.Sprintf(
		"path %q is already claimed by %s",
		err.Path, err.By.Src.String())
}
