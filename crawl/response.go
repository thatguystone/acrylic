package crawl

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
)

type crawlPath struct {
	ContentType string // Content-Type of path
	Path        string // Path where content lives
}

const (
	pathContentType = "application/x-acrylic-path"
	variantHeader   = "X-Acrylic-Variant"
)

// ServeFile is like http.ServeFile, except that if the requester is acrylic's
// crawler, it sends back the file's path rather than reading it into memory.
func ServeFile(w http.ResponseWriter, r *http.Request, path string) {
	if !strings.Contains(r.Header.Get("Accept"), pathContentType) {
		http.ServeFile(w, r, path)
		return
	}

	contType := mime.TypeByExtension(filepath.Ext(path))
	if contType == "" {
		contType = DefaultType
	}

	w.Header().Set("Content-Type", pathContentType)
	json.NewEncoder(w).Encode(crawlPath{
		ContentType: contType,
		Path:        path,
	})
}

// Variant sets the output name (and query string, if included) for the current
// request. This is mainly useful for handlers that change their output based on
// query string vals (eg. image scalers, css themes, etc).
//
// All links that point to original URL will be rewritten.
func Variant(w http.ResponseWriter, name string) {
	w.Header().Set(variantHeader, name)
}

type response struct {
	status int
	header http.Header
	body   responseBody
}

func newResponse(rr *httptest.ResponseRecorder) (*response, error) {
	resp := response{
		status: rr.Code,
		header: rr.HeaderMap,
	}

	contType := rr.HeaderMap.Get("Content-Type")

	if contType != pathContentType {
		resp.body.b = rr.Body.Bytes()
	} else {
		var cp crawlPath
		err := json.NewDecoder(rr.Body).Decode(&cp)
		if err != nil {
			return nil, err
		}

		contType = cp.ContentType
		resp.body.symSrc = cp.Path
	}

	if contType != "" {
		mediaType, _, err := mime.ParseMediaType(contType)
		if err != nil {
			return nil, err
		}

		resp.body.mediaType = mediaType
	}

	return &resp, nil
}

type responseBody struct {
	mediaType string // Parsed Content-Type
	symSrc    string // Path to original file
	b         []byte // If symSrc == ""
}

func (body *responseBody) canSymlink() bool {
	return body.symSrc != ""
}

func (body *responseBody) set(b []byte) {
	body.symSrc = ""
	body.b = b
}

func (body *responseBody) get() ([]byte, error) {
	if !body.canSymlink() {
		return body.b, nil
	}

	f, err := os.Open(body.symSrc)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return ioutil.ReadAll(f)
}

func (body *responseBody) reader() (io.ReadCloser, error) {
	if !body.canSymlink() {
		return ioutil.NopCloser(bytes.NewReader(body.b)), nil
	}

	return os.Open(body.symSrc)
}
