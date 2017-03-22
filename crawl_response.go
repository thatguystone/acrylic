package acrylic

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/pkg/errors"
)

const pathContentType = "application/" + Accept

type crawlPath struct {
	ContentType string
	Path        string
}

// Path is used to send a local path to the crawler rather than reading a file
// into memory and sending that back. This is mainly for cases where you need to
// send back a cached video, scaled image, etc.
func Path(w http.ResponseWriter, contentType, path string) {
	w.Header().Set("Content-Type", pathContentType)
	json.NewEncoder(w).Encode(crawlPath{
		ContentType: contentType,
		Path:        path,
	})
}

type response struct {
	contType string
	body     struct {
		buff *bytes.Buffer
		path string
	}
}

func newResponse(resp *httptest.ResponseRecorder) (r response, err error) {
	contType := resp.HeaderMap.Get("Content-Type")
	if contType != pathContentType {
		r.body.buff = resp.Body
	} else {
		var cp crawlPath
		err = json.NewDecoder(resp.Body).Decode(&cp)
		if err != nil {
			return
		}

		contType = cp.ContentType
		r.body.path = cp.Path
	}

	if contType != "" {
		r.contType, _, err = mime.ParseMediaType(contType)
		if err != nil {
			err = errors.Wrapf(err, "invalid Content-Type `%s`", contType)
			return
		}
	}

	return
}

func (r response) getBody() (io.ReadCloser, error) {
	if r.body.buff != nil {
		return ioutil.NopCloser(bytes.NewReader(r.body.buff.Bytes())), nil
	}

	return os.Open(r.body.path)
}

// saveTo saves the response body to the given destination. Nothing is written
// if the dest is the src (from a crawlPath response).
func (r response) saveTo(cr *Crawl, dst string) error {
	if r.body.buff != nil {
		return cr.save(dst, bytes.NewReader(r.body.buff.Bytes()))
	}

	bodyInfo, err := os.Stat(r.body.path)
	if err != nil {
		return err
	}

	dstInfo, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if os.SameFile(bodyInfo, dstInfo) {
		cr.setUsed(dst)
		return nil
	}

	f, err := os.Open(r.body.path)
	if err == nil {
		defer f.Close()
		err = cr.save(dst, f)
	}

	return err
}
