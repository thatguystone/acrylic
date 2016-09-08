package acrylic

import (
	"log"
	"net/http"

	"github.com/thatguystone/acrylic/internal/crawl"
)

// A Site contains configuration options used for a site
type Site struct {
	Handler     http.Handler // Handler to crawl
	EntryPoints []string     // Entry points to crawl
	Output      string       // Build directory
}

// Serve the site on the given addr
func (s *Site) Serve(addr string) {
	setDebug()

	log.Fatal(http.ListenAndServe(addr, s.Handler))
}

// Proxy runs a watching proxy server that rebuilds app any time a "*.go" file
// changes in any watched dir.
func (s *Site) Proxy(args ProxyArgs) {
	setDebug()

	p, err := newProxy(args)
	if err == nil {
		err = p.run()
	}

	log.Fatal(err)
}

// Build crawls the site, putting all found files into the Output directory.
func (s *Site) Build() {
	setProduction()

	crawl.Run(crawl.Args{
		Handler:     s.Handler,
		EntryPoints: s.EntryPoints,
		Output:      s.Output,
	})
}

// Templates creates a new Pongo2 TemplateSet that can be used to serve
// templates directly.
func (s *Site) Templates(root string) TemplateSet {
	return templates(root)
}

// ImgHandler creates a new image proxy that searches for images by URL,
// rooted at the given root.
func (s *Site) ImgHandler(root string) http.Handler {
	return newImgHandler(s, root)
}

// ScssHandler creates a new handler that serves css compiled from the args.
func (s *Site) ScssHandler(args ScssArgs) http.Handler {
	return newScssHandler(args)
}

// WebpackHandler creates a handler that serves the named webpack asset
func (s *Site) WebpackHandler(asset string) http.Handler {
	return newWebpackHandler(asset)
}
