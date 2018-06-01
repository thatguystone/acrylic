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

type body struct {
	contType  string // Original Content-Type
	mediaType string
	buff      *bytes.Buffer
	symSrc    string
}

func newBody(resp *httptest.ResponseRecorder) (*body, error) {
	b := body{
		contType: resp.HeaderMap.Get("Content-Type"),
	}

	if b.contType != pathContentType {
		b.buff = resp.Body
	} else {
		var cp crawlPath
		err := json.NewDecoder(resp.Body).Decode(&cp)
		if err != nil {
			return nil, err
		}

		b.contType = cp.ContentType
		b.symSrc = cp.Path
	}

	if b.contType != "" {
		mediaType, _, err := mime.ParseMediaType(b.contType)
		if err != nil {
			return nil, err
		}

		b.mediaType = mediaType
	}

	return &b, nil
}

func (b *body) setContent(s []byte) {
	b.symSrc = ""
	b.buff = bytes.NewBuffer(s)
}

func (b *body) getContent() ([]byte, error) {
	if b.canSymlink() {
		f, err := os.Open(b.symSrc)
		if err != nil {
			return nil, err
		}

		defer f.Close()
		return ioutil.ReadAll(f)
	}

	return b.buff.Bytes(), nil
}

func (b *body) getReader() (io.ReadCloser, error) {
	if b.canSymlink() {
		return os.Open(b.symSrc)
	}

	return ioutil.NopCloser(bytes.NewReader(b.buff.Bytes())), nil
}

func (b *body) canSymlink() bool {
	return b.symSrc != ""
}
