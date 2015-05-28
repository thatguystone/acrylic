package toner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/rainycape/vfs"
)

// Single-use, for generating a site
type site struct {
	cfg *Config
	fs  vfs.VFS
	d   data
	c   []content
	l   layouts
	k   kinds
	t   tags
}

type kinds struct {
}

type kind struct {
}

type tags struct {
}

type tag struct {
}

type file struct {
	srcPath string
	dstPath string
	info    os.FileInfo
}

type data map[string]interface{}

const createFlags = os.O_RDWR | os.O_CREATE | os.O_TRUNC

func newSite(cfg *Config, fs vfs.VFS) *site {
	s := &site{
		cfg: cfg,
		fs:  fs,
	}

	s.l.s = s

	return s
}

func (s *site) build() error {
	err := s.loadData()
	if err != nil {
		return err
	}

	err = s.l.check()
	if err != nil {
		return err
	}

	err = s.loadContent()
	if err != nil {
		return err
	}

	return s.generate()
}

func (s *site) loadContent() error {
	wg := sync.WaitGroup{}
	procs := runtime.GOMAXPROCS(-1)

	fileCh := make(chan file, procs*2)
	contentCh := make(chan content, procs*2)
	errCh := make(chan error)

	reader := func() {
		defer wg.Done()
		for f := range fileCh {
			c := content{
				s: s,
				f: f,
			}
			c.err = c.preprocess()

			contentCh <- c
		}
	}

	collector := func() {
		errors := []string{}

		for c := range contentCh {
			if c.err != nil {
				errors = append(errors, fmt.Sprintf("in %s: %s",
					c.f.srcPath,
					c.err.Error()))
				s.c = nil
			}

			if len(errors) == 0 {
				s.c = append(s.c, c)
			}
		}

		if len(errors) == 0 {
			errCh <- nil
		} else {
			errCh <- fmt.Errorf("errors while reading content: \n\t%s",
				strings.Join(errors, "\n\t"))
		}
	}

	for i := 0; i < procs; i++ {
		wg.Add(1)
		go reader()
	}

	go collector()

	err := s.walkRoot(s.cfg.ContentDir, func(f file) error {
		fileCh <- f
		return nil
	})

	close(fileCh)
	wg.Wait()
	close(contentCh)

	err2 := <-errCh
	if err == nil {
		err = err2
	} else {
		err = fmt.Errorf("%s\n%s", err, err2)
	}

	return err
}

func (s *site) generate() error {
	wg := sync.WaitGroup{}
	procs := runtime.GOMAXPROCS(-1)

	contentCh := make(chan content, procs*2)
	errCh := make(chan []string, procs*2)

	generator := func() {
		defer wg.Done()

		errors := []string{}
		for c := range contentCh {
			err := c.render()
			if err != nil {
				errors = append(errors, fmt.Sprintf("in %s: %s",
					c.f.srcPath,
					err.Error()))
			}
		}

		errCh <- errors
	}

	for i := 0; i < procs; i++ {
		wg.Add(1)
		go generator()
	}

	for _, c := range s.c {
		contentCh <- c
	}

	close(contentCh)
	wg.Wait()
	close(errCh)

	errors := []string{}
	for e := range errCh {
		errors = append(errors, e...)
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to generate site: \n\t%s",
			strings.Join(errors, "\n\t"))
	}

	return nil
}

func (s *site) walkRoot(p string, cb func(file) error) error {
	if !s.dexists(p) {
		return nil
	}

	var walk func(string, func(file) error) error
	walk = func(p string, cb func(file) error) error {
		dir, err := s.fs.ReadDir(p)
		if err != nil {
			return err
		}

		for _, info := range dir {
			p := filepath.Join(p, info.Name())

			if info.IsDir() {
				err = walk(p, cb)
			} else {
				err = cb(file{
					srcPath: p,
					info:    info,
				})
			}

			if err != nil {
				return err
			}
		}

		return nil
	}

	return walk(p, cb)
}

func (s *site) loadData() error {
	d := data{}

	err := s.walkRoot(s.cfg.DataDir, func(f file) error {
		return nil
	})

	s.d = d
	return err
}

func (s *site) fcreate(path string) (vfs.WFile, error) {
	return fcreate(s.fs, path, createFlags, s.cfg.FileMode)
}

func (s *site) fexists(path string) bool {
	info, err := s.fs.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func (s *site) dexists(path string) bool {
	info, err := s.fs.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
