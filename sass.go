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

type SassConfig struct {
	Entries      []string                     // Top-level files to build
	IncludePaths []string                     // Search directories for imports
	Logf         func(string, ...interface{}) // Where to log messages
}

type sass struct {
	cfg     SassConfig
	changed chan struct{}

	rwmtx      sync.RWMutex
	compiled   []byte
	compileErr error
	lastMod    *time.Time
}

func NewSass(cfg SassConfig) HandlerWatcher {
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}

	s := sass{
		cfg:     cfg,
		changed: make(chan struct{}, 1),
	}

	// Lock, pending first build
	s.rwmtx.Lock()
	go s.run()

	s.changed <- struct{}{}

	return &s
}

func (s *sass) Changed(evs WatchEvents) {
	if evs.HasExt(".scss") {
		s.changed <- struct{}{}
	}
}

func (s *sass) run() {
	first := true

	for range s.changed {
		if !first {
			s.rwmtx.Lock()
		}
		first = false

		start := time.Now()
		s.cfg.Logf("I: sass %s: rebuilding...\n", s.cfg.Entries)
		s.compileErr = s.rebuild()

		s.rwmtx.Unlock()

		if s.compileErr == nil {
			s.cfg.Logf("I: sass %s: rebuild took %s\n",
				s.cfg.Entries, time.Since(start))
		} else {
			s.cfg.Logf("E: sass %s: rebuild failed:\n%v",
				s.cfg.Entries, stringc.Indent(s.compileErr.Error(), crawl.ErrIndent))
		}
	}
}

func (s *sass) rebuild() error {
	s.compiled = nil
	s.compileErr = nil
	s.lastMod = nil

	var imports []string
	var buff bytes.Buffer

	for _, path := range s.cfg.Entries {
		imports = append(imports, path)

		comp, err := libsass.New(&buff, nil,
			libsass.Path(path),
			libsass.IncludePaths(s.cfg.IncludePaths),

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

func (s *sass) getLastMod() time.Time {
	if s.lastMod != nil {
		return *s.lastMod
	}

	return time.Now()
}

// ServeHTTP implements http.Handler
func (s *sass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.rwmtx.RLock()
	defer s.rwmtx.RUnlock()

	if s.compileErr != nil {
		HTTPError(w, s.compileErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	setMustRevalidate(w)
	http.ServeContent(
		w, r, "",
		s.getLastMod(), bytes.NewReader(s.compiled))
}
