package acrylic

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wellington/go-libsass"
)

type ScssArgs struct {
	Entry        string   // Main entry point
	Recurse      string   // Directory to search for additional files
	IncludePaths []string // Include search path
}

type scssHandler struct {
	ScssArgs
	handler

	rwmtx   sync.RWMutex
	imports []string // Set by compiler
}

func newScssHandler(args ScssArgs) *scssHandler {
	h := &scssHandler{
		ScssArgs: args,
	}

	h.Entry = filepath.Clean(h.Entry)
	h.Recurse = filepath.Clean(h.Recurse)

	return h
}

func (h *scssHandler) compile() (b bytes.Buffer, lastMod time.Time, err error) {
	h.rwmtx.Lock()
	defer h.rwmtx.Unlock()

	entries := []string{h.Entry}

	style := libsass.NESTED_STYLE
	if !isDebug() {
		style = libsass.COMPRESSED_STYLE
	}

	if h.Recurse != "" {
		err = filepath.Walk(h.Recurse,
			func(path string, info os.FileInfo, err error) error {
				if !h.shouldInclude(path) {
					return nil
				}

				for _, e := range entries {
					if e == path {
						return nil
					}
				}

				entries = append(entries, path)

				return nil
			})
		if err != nil {
			return
		}
	}

	h.imports = nil

	for _, f := range entries {
		comp, cErr := libsass.New(&b, nil,
			libsass.Path(f),
			libsass.IncludePaths(h.IncludePaths),
			libsass.OutputStyle(style))

		if cErr == nil {
			cErr = comp.Run()
		}

		if cErr != nil {
			err = fmt.Errorf("in file %s: %v", f, cErr)
			return
		}

		h.imports = append(h.imports, comp.Imports()...)
	}

	lastMod = h.getLastModified()

	return
}

// shouldInclude checks if the given path should be included in a build.
func (h *scssHandler) shouldInclude(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)

	if ext != ".scss" {
		return false
	}

	if strings.HasPrefix(base, "_") {
		return false
	}

	if h.Recurse == "" {
		return false
	}

	if strings.HasPrefix(path, h.Recurse) {
		return true
	}

	return false
}

func (h *scssHandler) getLastModified() (lastMod time.Time) {
	for _, imp := range h.imports {
		info, err := os.Stat(imp)
		if err != nil {
			log.Printf("E: [scss] failed to update modification time of: %s: %v",
				imp, err)
			return
		}

		mod := info.ModTime()
		if mod.After(lastMod) {
			lastMod = mod
		}
	}

	return
}

func (h *scssHandler) checkModified(
	w http.ResponseWriter,
	r *http.Request) bool {

	h.rwmtx.RLock()
	defer h.rwmtx.RUnlock()

	return h.handler.checkModified(h.getLastModified(), w, r)
}

func (h *scssHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.checkModified(w, r) {
		return
	}

	b, lastMod, err := h.compile()
	if err != nil {
		h.errorf(w, err, "[scss] compile failed")
	} else {
		w.Header().Set("Content-Type", "text/css")
		h.setLastModified(lastMod, w)
		b.WriteTo(w)
	}
}
