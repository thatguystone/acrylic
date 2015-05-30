package toner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rainycape/vfs"
	"github.com/tdewolff/minify"
	minhtml "github.com/tdewolff/minify/html"
)

// Single-use, for generating a site
type site struct {
	cfg *Config
	fs  vfs.VFS
	min *minify.Minify
	d   data
	c   []content
	l   layouts
	t   tags
	a   assets
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
		min: minify.New(),
	}

	s.min.AddFunc("text/html", minhtml.Minify)

	s.l.s = s
	s.a.s = s

	return s
}

func (s *site) build() error {
	err := s.loadData()
	if err != nil {
		return err
	}

	err = s.l.init()
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

	fileCh := make(chan file, s.cfg.Jobs*2)
	contentCh := make(chan content, s.cfg.Jobs*2)
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
		errs := []string{}

		for c := range contentCh {
			if c.err != nil {
				errs = append(errs, fmt.Sprintf("in %s: %s",
					c.f.srcPath,
					c.err.Error()))
				s.c = nil
			}

			if len(errs) == 0 {
				s.c = append(s.c, c)
			}
		}

		if len(errs) == 0 {
			errCh <- nil
		} else {
			errCh <- fmt.Errorf("errors while reading content: \n\t%s",
				strings.Join(errs, "\n\t"))
		}
	}

	for i := uint(0); i < s.cfg.Jobs; i++ {
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

	contentCh := make(chan content, s.cfg.Jobs*2)
	errCh := make(chan []string, s.cfg.Jobs)

	generator := func() {
		defer wg.Done()

		errs := []string{}
		for c := range contentCh {
			err := c.render()
			if err != nil {
				errs = append(errs, fmt.Sprintf("in %s: %s",
					c.f.srcPath,
					err.Error()))
			}
		}

		errCh <- errs
	}

	for i := uint(0); i < s.cfg.Jobs; i++ {
		wg.Add(1)
		go generator()
	}

	for _, c := range s.c {
		contentCh <- c
	}

	close(contentCh)
	wg.Wait()
	close(errCh)

	errs := []string{}
	for e := range errCh {
		errs = append(errs, e...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to generate site: \n\t%s",
			strings.Join(errs, "\n\t"))
	}

	return nil
}

func (s *site) walkRoot(p string, cb func(file) error) error {
	if !s.dExists(p) {
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

func (s *site) fCreate(path string) (vfs.WFile, error) {
	return fCreate(s.fs, path, createFlags, s.cfg.FileMode)
}

func (s *site) fExists(path string) bool {
	info, err := s.fs.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func (s *site) dExists(path string) bool {
	info, err := s.fs.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
