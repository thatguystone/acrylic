package acrylic

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// CacheBustParam is the query string parameter used for cache busters
const CacheBustParam = "v"

type handler struct{}

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

func (h handler) checkModified(
	lastMod time.Time,
	w http.ResponseWriter,
	r *http.Request) bool {

	if lastMod.IsZero() {
		return false
	}

	t, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err == nil && lastMod.Before(t.Add(time.Second)) {
		w.Header().Del("Content-Type")
		w.Header().Del("Content-Length")
		w.WriteHeader(http.StatusNotModified)
		return true
	}

	return false
}

func (h handler) setLastModified(lastMod time.Time, w http.ResponseWriter) {
	w.Header().Set("Last-Modified", lastMod.UTC().Format(http.TimeFormat))
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
