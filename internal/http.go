package internal

import (
	"fmt"
	"html"
	"net/http"
)

// HTTPError sends a human-readable HTTP error page
func HTTPError(w http.ResponseWriter, err string, code int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	fmt.Fprintf(w, ""+
		"<style>\n"+
		"	body {\n"+
		"		background: #272822;\n"+
		"		color: #fff;\n"+
		"	}\n"+
		"</style>\n"+
		"<h1>Error</h1>\n"+
		"<pre>%s</pre>\n",
		html.EscapeString(err))
}

// SetMustRevalidate sets a response's cache-control to must-revalidate
func SetMustRevalidate(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "must-revalidate, max-age=0")
}
