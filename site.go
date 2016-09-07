package acrylic

import (
	"log"
	"net/http"
)

// A Site contains configuration options used for a site
type Site struct {
	Handler     http.Handler // Handler to crawl
	EntryPoints []string     // Entry points to crawl
	Output      string       // Build directory
}

// Serve the site on the given addr
func (s *Site) Serve(addr string) {
	log.Fatal(http.ListenAndServe(addr, s.Handler))
}

// Run a debug proxy
func (s *Site) Proxy(args ProxyArgs) {
	p, err := newProxy(args)
	if err == nil {
		err = p.run()
	}

	log.Fatal(err)
}

func (s *Site) Build() {
	s.build() // See: build.go
}

func (s *Site) Templates(root string) TemplateSet {
	return templates(root)
}

func (s *Site) ImgHandler(root string) http.Handler {
	return newImgHandler(s, root)
}

func (s *Site) ScssHandler(args ScssArgs) http.Handler {
	return newScssHandler(args)
}

func (s *Site) WebpackHandler(asset string) http.Handler {
	return newWebpackHandler(asset)
}
