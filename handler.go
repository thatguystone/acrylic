package acrylic

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

// CacheBustParam is the query string parameter used for cache busters
const CacheBustParam = "v"

type handler struct{}

func (h handler) needsBusted(r *http.Request) bool {
	return !isDebug() && r.FormValue(CacheBustParam) == ""
}

func (h handler) redirectBusted(
	w http.ResponseWriter, r *http.Request,
	url url.URL, buster string) {

	q := url.Query()
	q.Set(CacheBustParam, buster)
	url.RawQuery = q.Encode()

	http.Redirect(w, r, url.String(), http.StatusFound)
}

func (h handler) errorf(
	w http.ResponseWriter,
	err error,
	msg string, args ...interface{}) {

	msg = fmt.Sprintf(msg, args...)

	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "%s: %v", msg, err)
	log.Printf("E: %s: %v", msg, err)
}

func (h handler) invalidf(
	w http.ResponseWriter,
	err error,
	msg string, args ...interface{}) {

	msg = fmt.Sprintf(msg, args...)

	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "%s: %v", msg, err)
}

func (h handler) hashBuster(b []byte) string {
	sum := sha1.Sum(b)
	return base64.URLEncoding.EncodeToString(sum[:])[:12]
}

func (h handler) statFile(
	path string,
	w http.ResponseWriter,
	mustExist bool) (os.FileInfo, bool, error) {

	stat, err := os.Stat(path)

	exists := err == nil && !stat.IsDir()
	if exists {
		return stat, true, nil
	}

	if os.IsNotExist(err) {
		err = nil
	}

	if err != nil {
		h.errorf(w, err, "E: failed to stat %s", path)
	} else if mustExist {
		w.WriteHeader(http.StatusNotFound)
	}

	return nil, false, err
}
