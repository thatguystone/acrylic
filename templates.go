package acrylic

import (
	"log"
	"net/http"

	"github.com/flosch/pongo2"
	"github.com/julienschmidt/httprouter"
	"github.com/thatguystone/cog"
)

// TemplateSet is a wrapper around Pongo2's TemplateSet, providing additional
// functionality.
type TemplateSet struct {
	*pongo2.TemplateSet
}

func templates(root string) (ts TemplateSet) {
	loader := pongo2.MustNewLocalFileSystemLoader(root)

	ts.TemplateSet = pongo2.NewSet("set", loader)
	ts.Debug = isDebug()

	return
}

// Handler creates a new handler that serves the template at the given path
// (relative to the TemplateSet's root).
func (ts TemplateSet) Handler(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := ts.FromCache(path)
		cog.Must(err, "failed to find template: %s", path)

		err = tmpl.ExecuteWriter(nil, w)
		if err != nil {
			log.Printf("E: [tmpl] failed to write %s to client: %v", path, err)
		}
	})
}

// Handle creates a new handle that serves the template at the given path
// (relative to the TemplateSet's root).
func (ts TemplateSet) Handle(path string) httprouter.Handle {
	handler := ts.Handler(path)

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		handler.ServeHTTP(w, r)
	}
}
