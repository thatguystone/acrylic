package acrylic

import (
	"bytes"
	"net/http"
)

type scssHandler struct {
	handler
	c scss
}

func newScssHandler(args ScssArgs) *scssHandler {
	h := &scssHandler{}
	h.c.init(args)
	return h
}

func (h *scssHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, lastMod, err := h.c.pollChanges()
	switch {
	case err != nil:
		h.errorf(w, err, "[scss] compile failed")

	case h.needsBusted(r):
		h.handler.redirectBusted(
			w, r,
			*r.URL, h.hashBuster(body))

	default:
		w.Header().Set("Content-Type", "text/css")
		http.ServeContent(w, r, "", lastMod, bytes.NewReader(body))
	}
}
