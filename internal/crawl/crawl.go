package crawl

import (
	"log"
	"net/http"
)

// Args used to crawl the site
type Args struct {
	Handler     http.Handler // Handler to crawl
	EntryPoints []string     // Entry points to crawl
	Output      string       // Build directory
	Logf        func(string, ...interface{})
}

// Run runs the crawl on the given Handler
func Run(args Args) {
	if len(args.EntryPoints) == 0 {
		args.EntryPoints = []string{"/"}
	}

	if args.Logf == nil {
		args.Logf = log.Printf
	}

	newState(args).crawl()
}
