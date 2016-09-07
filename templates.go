package acrylic

import (
	"log"
	"net/http"

	"github.com/flosch/pongo2"
	"github.com/julienschmidt/httprouter"
	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/cio"
)

type TemplateSet struct {
	*pongo2.TemplateSet
}

func templates(root string) (ts TemplateSet) {
	loader := pongo2.MustNewLocalFileSystemLoader(root)

	ts.TemplateSet = pongo2.NewSet("set", loader)
	ts.Debug = isDebug()

	return
}

func (ts TemplateSet) Handler(path string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		tmpl, err := ts.FromCache(path)
		cog.Must(err, "failed to find template: %s", path)

		wc := cio.NopWriteCloser(w)
		if !isDebug() {
			wc = Minify.Writer("text/html", w)
		}

		err = tmpl.ExecuteWriter(nil, wc)
		if err == nil {
			err = wc.Close()
		}

		if err != nil {
			log.Printf("E: [tmpl] failed to write %s to client: %v", path, err)
		}
	}
}
