package acrylic

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/stringc"
	"github.com/wellington/go-libsass"
)

// Sass compiles scss files
type Sass struct {
	Entries      []string // Top-level files to build
	Recurse      []string // Directories to recursively search for *.scss files
	IncludePaths []string // Search directories for imports

	once    sync.Once
	changed chan struct{}

	rwmtx    sync.RWMutex
	compiled bytes.Buffer // Compiled output
	lastMod  time.Time
	err      error
}

func (s *Sass) init() (first bool) {
	s.once.Do(func() {
		first = true

		for i, recurse := range s.Recurse {
			path, err := filepath.Abs(recurse)
			cog.Must(err, "failed to get abspath of %s", recurse)

			s.Recurse[i] = path
		}

		s.changed = make(chan struct{}, 2)

		s.rwmtx.Lock() // Lock, pending first build
		s.changed <- struct{}{}

		go s.run()
	})

	return
}

func (s *Sass) run() {
	first := true

	for range s.changed {
		if !first {
			s.rwmtx.Lock()
		}
		first = false

		s.err = s.rebuild()
		s.rwmtx.Unlock()

		if s.err != nil {
			log.Printf("[sass] rebuild failed:\n%v",
				stringc.Indent(s.err.Error(), indent))
		}
	}
}

func (s *Sass) rebuild() error {
	s.compiled.Reset()
	s.lastMod = time.Time{}
	entries := s.Entries

	for _, recurse := range s.Recurse {
		err := filepath.Walk(recurse,
			func(path string, info os.FileInfo, err error) error {
				if s.shouldInclude(path) {
					entries = append(entries, path)
				}

				return err
			})
		if err != nil {
			return errors.Wrap(err, "filepath.Walk")
		}
	}

	for _, f := range entries {
		comp, err := libsass.New(&s.compiled, nil,
			libsass.Path(f),
			libsass.IncludePaths(s.IncludePaths),

			// Default to Nested: it's Crawl's job to compress
			libsass.OutputStyle(libsass.NESTED_STYLE))

		if err == nil {
			err = comp.Run()
		}

		if err == nil {
			imports := comp.Imports()
			imports = append(imports, f)
			err = s.updateLastMod(imports)
		}

		if err != nil {
			return errors.Wrapf(err, "in file %s: %v", f, err)
		}
	}

	return nil
}

func (s *Sass) updateLastMod(paths []string) error {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if info.ModTime().After(s.lastMod) {
			s.lastMod = info.ModTime()
		}
	}

	return nil
}

// shouldInclude checks if the given path should be included in a build.
func (s *Sass) shouldInclude(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return ext == ".scss" && !strings.HasPrefix(base, "_")
}

// Changed implements Watcher
func (s *Sass) Changed(evs WatchEvents) {
	changed := false
	for _, ev := range evs {
		if s.shouldInclude(ev.Path()) {
			changed = true
			break
		}
	}

	if !s.init() && changed {
		s.changed <- struct{}{}
	}
}

// ServeHTTP implements http.Handler
func (s *Sass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.init()

	s.rwmtx.RLock()
	defer s.rwmtx.RUnlock()

	switch {
	case s.err != nil:
		sendError(s.err, w)

	default:
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		http.ServeContent(
			w, r, "",
			s.lastMod, bytes.NewReader(s.compiled.Bytes()))
	}
}
