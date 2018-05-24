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
)

type crawlPath struct {
	ContentType string
	Path        string
}

// PathContentType is sent in the Accept header when the crawler makes a
// request for a resource.
const PathContentType = "application/x-acrylic-path"

// Path is used to send a local path to the crawler rather than reading a file
// into memory and sending that back. This is mainly for cases where you need to
// send back a cached video, scaled image, etc.
func Path(w http.ResponseWriter, contentType, path string) {
	w.Header().Set("Content-Type", PathContentType)
	json.NewEncoder(w).Encode(crawlPath{
		ContentType: contentType,
		Path:        path,
	})
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

	if b.contType != PathContentType {
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
