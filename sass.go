package acrylic

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/thatguystone/acrylic/crawl"
	"github.com/thatguystone/cog/stringc"
	"github.com/wellington/go-libsass"
)

// Sass compiles scss files
type Sass struct {
	Entries      []string                     // Top-level files to build
	IncludePaths []string                     // Search directories for imports
	Concat       []string                     // Extra files to concat to the output
	Logf         func(string, ...interface{}) // Where to log messages

	changed chan struct{}

	rwmtx      sync.RWMutex
	compiled   []byte
	compileErr error
	lastMod    *time.Time
}

func (s *Sass) Start(*Watch) {
	if s.Logf == nil {
		s.Logf = log.Printf
	}

	s.changed = make(chan struct{}, 1)
	s.changed <- struct{}{}

	// Lock, pending first build
	s.rwmtx.Lock()

	go s.run()
}

func (s *Sass) Changed(evs WatchEvents) {
	if evs.HasExt(".scss") {
		s.changed <- struct{}{}
	}
}

func (s *Sass) run() {
	first := true

	for range s.changed {
		if !first {
			s.rwmtx.Lock()
		}
		first = false

		start := time.Now()
		s.Logf("I: sass %s: rebuilding...\n", s.Entries)
		s.compileErr = s.rebuild()

		s.rwmtx.Unlock()

		if s.compileErr == nil {
			s.Logf("I: sass %s: rebuild took %s\n",
				s.Entries, time.Since(start))
		} else {
			s.Logf("E: sass %s: rebuild failed:\n%v",
				s.Entries, stringc.Indent(s.compileErr.Error(), crawl.ErrIndent))
		}
	}
}

func (s *Sass) rebuild() error {
	s.compiled = nil
	s.compileErr = nil
	s.lastMod = nil

	var imports []string
	var buff bytes.Buffer

	for _, path := range s.Entries {
		imports = append(imports, path)

		comp, err := libsass.New(&buff, nil,
			libsass.Path(path),
			libsass.IncludePaths(s.IncludePaths),

			// Default to Nested: it's Crawl's job to compress
			libsass.OutputStyle(libsass.NESTED_STYLE))

		if err != nil {
			return err
		}

		err = comp.Run()
		if err != nil {
			return err
		}

		imports = append(imports, comp.Imports()...)
	}

	s.compiled = buff.Bytes()

	for _, path := range imports {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		mod := info.ModTime()
		if s.lastMod == nil || mod.After(*s.lastMod) {
			s.lastMod = &mod
		}
	}

	return nil
}

func (s *Sass) getLastMod() time.Time {
	if s.lastMod != nil {
		return *s.lastMod
	}

	return time.Now()
}

// ServeHTTP implements http.Handler
func (s *Sass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.rwmtx.RLock()
	defer s.rwmtx.RUnlock()

	if s.compileErr != nil {
		HTTPError(w, s.compileErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	setCacheHeaders(w)
	http.ServeContent(
		w, r, "",
		s.getLastMod(), bytes.NewReader(s.compiled))
}
