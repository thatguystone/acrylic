// Package sass implements a sass compiler
package sass

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/thatguystone/acrylic"
	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/acrylic/watch"
	libsass "github.com/wellington/go-libsass"
)

type sass struct {
	entries      []string
	includePaths []string
	log          acrylic.Logger
	changed      chan struct{}

	rwmtx      sync.RWMutex
	compiled   []byte
	compileErr error
	lastMod    *time.Time
}

// New creates a new sass compiler
func New(entry string, opts ...Option) http.Handler {
	s := &sass{
		entries: []string{entry},
		log:     internal.NewLogger(fmt.Sprintf("sass{%s}", entry), log.Printf),
		changed: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt.applyTo(s)
	}

	// Lock, pending first build
	s.rwmtx.Lock()
	go s.run()

	s.changed <- struct{}{}

	return s
}

func (s *sass) Changed(evs watch.Events) {
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
		s.log.Log("rebuilding...")
		s.compileErr = s.rebuild()

		s.rwmtx.Unlock()

		if s.compileErr == nil {
			s.log.Log(fmt.Sprintf("rebuild took %s", time.Since(start)))
		} else {
			s.log.Error(s.compileErr, "rebuild failed")
		}
	}
}

func (s *sass) rebuild() error {
	s.compiled = nil
	s.compileErr = nil
	s.lastMod = nil

	var imports []string
	var buff bytes.Buffer

	for _, path := range s.entries {
		imports = append(imports, path)

		comp, err := libsass.New(&buff, nil,
			libsass.Path(path),
			libsass.IncludePaths(s.includePaths),

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
		internal.HTTPError(
			w, s.compileErr.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	internal.SetMustRevalidate(w)
	http.ServeContent(
		w, r, "",
		s.getLastMod(), bytes.NewReader(s.compiled))
}
