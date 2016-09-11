package acrylic

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wellington/go-libsass"
)

// ScssArgs collects the arguments to pass to ScssHandler().
type ScssArgs struct {
	Entry        string   // Main entry point
	Recurse      []string // Directories to search for additional files
	IncludePaths []string // Include search path
}

// Wraps scss compilation
type scss struct {
	args ScssArgs

	rmtx    sync.RWMutex
	imports []string  // Set by compiler
	cached  []byte    // Cached sheet
	lastMod time.Time // Value to compare to return of pollLastMod
}

func (c *scss) init(args ScssArgs) {
	c.args = args
	c.args.Entry = filepath.Clean(c.args.Entry)
	for i, recurse := range c.args.Recurse {
		c.args.Recurse[i] = filepath.Clean(recurse)
	}
}

func (c *scss) recompile() error {
	entries := []string{c.args.Entry}

	for _, recurse := range c.args.Recurse {
		err := filepath.Walk(recurse,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if !c.shouldInclude(path) {
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
			return err
		}
	}

	c.imports = nil

	var b bytes.Buffer
	for _, f := range entries {
		comp, err := libsass.New(&b, nil,
			libsass.Path(f),
			libsass.IncludePaths(c.args.IncludePaths),
			libsass.OutputStyle(libsass.NESTED_STYLE))

		if err == nil {
			err = comp.Run()
		}

		if err != nil {
			return fmt.Errorf("in file %s: %v", f, err)
		}

		c.imports = append(c.imports, comp.Imports()...)
	}

	c.cached = b.Bytes()

	return nil
}

// shouldInclude checks if the given path should be included in a build.
func (c *scss) shouldInclude(path string) bool {
	base := filepath.Base(path)
	ext := filepath.Ext(base)

	if ext != ".scss" {
		return false
	}

	if strings.HasPrefix(base, "_") {
		return false
	}

	return true
}

func (c *scss) getLastMod() (lastMod time.Time, err error) {
	for _, imp := range c.imports {
		var info os.FileInfo

		info, err = os.Stat(imp)
		if err != nil {
			err = errors.Wrapf(err,
				"[scss] failed to update modification time of: %s",
				imp)
			return
		}

		mod := info.ModTime()
		if mod.After(lastMod) {
			lastMod = mod
		}
	}

	return
}

func (c *scss) pollChanges() (sheet []byte, lastMod time.Time, err error) {
	c.rmtx.RLock()
	defer c.rmtx.RUnlock()

	lastMod, err = c.getLastMod()
	if err != nil {
		return
	}

	changed := lastMod.IsZero() || !lastMod.Equal(c.lastMod)
	if changed {
		err = c.recompile()
	}

	if changed && err == nil {
		c.lastMod, err = c.getLastMod()
	}

	sheet = c.cached
	lastMod = c.lastMod
	return
}
