package acrylic

import (
	"fmt"
	"html"
	"net/http"
)

type HandlerWatcher interface {
	http.Handler
	Watcher
}

// ServeMux creates an http.ServeMux
type ServeMux map[string]HandlerWatcher

// Start implements Watcher. It calls w.Notify on all contained
// HandlerWatchers.
func (mux ServeMux) Start(w *Watch) {
	for _, handler := range mux {
		w.Notify(handler)
	}
}

// Changed implements Watcher
func (ServeMux) Changed(evs WatchEvents) {}

// MakeHandler turns a ServeMux into an http.Handler
func (mux ServeMux) MakeHandler() *http.ServeMux {
	smux := http.NewServeMux()
	for path, handler := range mux {
		smux.Handle(path, handler)
	}

	return smux
}

// type NoopHandler http.Handler
// func (NoopHandler) Changed([]WatchEvents) {}
// func (h NoopHandler) ServeHTTP() {}

func sendError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	fmt.Fprintf(w, ""+
		"<style>\n"+
		"	body {\n"+
		"		background: #252830;\n"+
		"		color: #fff;\n"+
		"	}\n"+
		"</style>\n"+
		"<h1>Error</h1>\n"+
		"<pre>%s</pre>\n",
		html.EscapeString(err.Error()))
}

func setCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "must-revalidate, max-age=0")
}
