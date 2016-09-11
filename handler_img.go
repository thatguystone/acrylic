package acrylic

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type imgHandler struct {
	handler
	s          *Site
	root       string
	fileServer http.Handler
}

func newImgHandler(s *Site, root string) imgHandler {
	return imgHandler{
		s:    s,
		root: root,
		fileServer: http.FileServer(staticDirs{
			root,
			s.Output,
		}),
	}
}

func (h imgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid query: %v", err)
		return
	}

	srcPath := filepath.Join(h.root, upath)

	im, err := newImg(srcPath, r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid args: %v", err)
		return
	}

	if !im.isFinalPath || (!isDebug() && r.FormValue(CacheBustParam) == "") {
		h.redirect(w, r, srcPath, r.URL.Path)
	} else {
		// Provides Last-Modified caching
		h.fileServer.ServeHTTP(w, r)
	}
}

func (h imgHandler) scale(
	w http.ResponseWriter, r *http.Request,
	srcPath string, im img) {

	srcStat, ok, _ := h.statFile(srcPath, w, true)
	if !ok {
		return
	}

	scaledName := im.scaledName()
	dstPath := filepath.Join(h.s.Output,
		filepath.Dir(r.URL.String()),
		scaledName)

	dstStat, ok, err := h.statFile(dstPath, w, false)
	if err != nil {
		return
	}

	// If the internal cache is outdated, update before serving
	if !ok || !srcStat.ModTime().Equal(dstStat.ModTime()) {
		err := im.scale(dstPath)
		if err != nil {
			h.errorf(w, err, "failed to scale img")
			return
		}

		err = os.Chtimes(dstPath, srcStat.ModTime(), srcStat.ModTime())
		if err != nil {
			h.errorf(w, err, "failed to update scaled img times")
			return
		}
	}

	h.redirect(w, r, srcPath, "./"+scaledName)
}

func (h imgHandler) redirect(
	w http.ResponseWriter, r *http.Request,
	srcPath string, toPath string) {

	srcStat, ok, _ := h.statFile(srcPath, w, true)
	if !ok {
		return
	}

	url := url.URL{
		Path: toPath,
		RawQuery: fmt.Sprintf("%s=%d",
			CacheBustParam,
			srcStat.ModTime().Unix()),
	}

	http.Redirect(w, r, url.String(), http.StatusFound)
}
