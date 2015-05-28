package toner

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/rainycape/vfs"
)

// Toner represents a single site
type Toner struct {
	cfg Config
	fs  vfs.VFS
}

type file struct {
	path string
	info os.FileInfo
}

type data map[string]interface{}

func newToner(cfg Config, fs vfs.VFS) *Toner {
	return &Toner{
		cfg: cfg,
		fs:  fs,
	}
}

func New(cfg Config) (*Toner, error) {
	if cfg.Root == "" {
		cfg.Root = "."
	}

	abs, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}

	var v vfs.VFS
	fs, err := vfs.FS(abs)
	if err == nil {
		v, err = vfs.Chroot(".", fs)
	}

	return newToner(cfg, v), nil
}

// Build builds the current site
func (t *Toner) Build() error {
	if err := t.cfg.reload(); err != nil {
		return err
	}

	d, err := t.loadData()
	if err != nil {
		return err
	}

	l, err := newLayouts(t, d)
	if err != nil {
		return err
	}

	c, err := t.loadContent()
	if err != nil {
		return err
	}

	return t.generate(c, d, l)
}

func (t *Toner) loadContent() ([]content, error) {
	wg := sync.WaitGroup{}
	procs := runtime.GOMAXPROCS(-1)

	contents := []content{}
	fileCh := make(chan file, procs*2)
	contentCh := make(chan content, procs*2)
	errCh := make(chan error)

	reader := func() {
		defer wg.Done()
		for f := range fileCh {
			contentCh <- t.readContent(f)
		}
	}

	collector := func() {
		errors := []string{}

		for c := range contentCh {
			if c.err != nil {
				errors = append(errors, c.err.Error())
				contents = nil
			}

			if len(errors) == 0 {
				contents = append(contents, c)
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

	err := t.walkRoot("/content", func(f file) error {
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

	return contents, err
}

func (t *Toner) readContent(f file) (c content) {
	c.f = f

	ff, err := t.fs.Open(f.path)
	if err != nil {
		c.err = err
		return
	}

	defer ff.Close()

	c.rawContent, err = ioutil.ReadAll(ff)
	if err != nil {
		c.err = err
		return
	}

	return
}

func (t *Toner) generate(cs []content, d data, l *layouts) error {
	wg := sync.WaitGroup{}
	procs := runtime.GOMAXPROCS(-1)

	contentCh := make(chan content, procs*2)
	errCh := make(chan []string, procs*2)

	generator := func() {
		defer wg.Done()

		errors := []string{}
		for c := range contentCh {
			err := t.generateContent(c)
			if err != nil {
				errors = append(errors, err.Error())
			}
		}

		errCh <- errors
	}

	for i := 0; i < procs; i++ {
		wg.Add(1)
		go generator()
	}

	for _, c := range cs {
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

func (t *Toner) generateContent(c content) error {
	return nil
}

func (t *Toner) walkRoot(p string, cb func(file) error) error {
	if !t.dexists(p) {
		return nil
	}

	var walk func(string, func(file) error) error
	walk = func(p string, cb func(file) error) error {
		dir, err := t.fs.ReadDir(p)
		if err != nil {
			return err
		}

		for _, info := range dir {
			p := filepath.Join(p, "/", info.Name())

			if info.IsDir() {
				err = walk(p, cb)
			} else {
				err = cb(file{
					path: p,
					info: info,
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

func (t *Toner) loadData() (data, error) {
	d := data{}

	err := t.walkRoot("/data", func(f file) error {
		return nil
	})

	return d, err
}

func (t *Toner) fexists(path string) bool {
	info, err := t.fs.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

func (t *Toner) dexists(path string) bool {
	info, err := t.fs.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
