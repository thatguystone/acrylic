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
	if !strings.HasPrefix(r.URL.Path, "/") {
		r.URL.Path = "/" + r.URL.Path
	}

	err := r.ParseForm()
	if err != nil {
		h.invalidf(w, err, "[img] invalid args")
		return
	}

	im, err := newImg(filepath.Join(h.root, r.URL.Path), r.Form)
	if err != nil {
		h.invalidf(w, err, "[img] invalid args")
		return
	}

	srcStat, ok, _ := h.statFile(im.srcPath, w, true)
	if !ok {
		return
	}

	switch {
	case !im.isFinalPath || h.needsBusted(r):
		h.handler.redirectBusted(
			w, r,
			url.URL{Path: "./" + im.resolvedName},
			fmt.Sprintf("%d", srcStat.ModTime().Unix()))

	default:
		h.scale(w, r, srcStat, im)
	}
}

func (h imgHandler) scale(
	w http.ResponseWriter, r *http.Request,
	srcStat os.FileInfo, im *img) {

	dstPath := filepath.Join(h.s.Output,
		filepath.Dir(r.URL.String()),
		im.resolvedName)

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

	// Provides Last-Modified caching
	h.fileServer.ServeHTTP(w, r)
}
