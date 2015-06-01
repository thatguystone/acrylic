package toner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	p2 "github.com/flosch/pongo2"
	"github.com/tdewolff/minify"
	minhtml "github.com/tdewolff/minify/html"
)

type site struct {
	cfg *Config
	min *minify.Minify
	mtx sync.Mutex
	cs  contents
	d   data
	l   map[string]*layout

	wg        sync.WaitGroup
	contentCh chan file

	stats BuildStats
	errs  errs
}

type data map[string]interface{}

var reservedPaths = []string{
	layoutPubDir,
	staticPubDir,
	themePubDir,
}

func newSite(cfg *Config) *site {
	s := &site{
		cfg: cfg,
		min: minify.New(),
		l:   map[string]*layout{},
	}

	s.cs.init(s)
	s.min.AddFunc("text/html", minhtml.Minify)

	return s
}

func (s *site) build() (BuildStats, []Error) {
	s.loadData()

	if !s.errs.has() {
		s.runContentReader()
		defer s.stopContentReader()

		s.loadLayouts()
	}

	if !s.errs.has() {
		s.loadContent()
	}

	s.stopContentReader()

	if !s.errs.has() {
		s.generate()
	}

	return s.stats, s.errs.s
}

func (s *site) addContent(f file, isContent bool) {
	if s.contentCh == nil {
		panic("attempt to add content when content readers aren't running")
	}

	if isContent {
		if res := fPathCheckFor(f.dstPath, reservedPaths...); res != "" {
			s.errs.add(f.srcPath, fmt.Errorf("use of reserved path `%s` is not allowed", res))
			return
		}
	}

	s.contentCh <- f
}

func (s *site) runContentReader() {
	if s.contentCh != nil {
		panic("attempt to run content readers when already running")
	}

	ch := make(chan file, s.cfg.Jobs*2)
	reader := func() {
		defer s.wg.Done()
		for f := range ch {
			err := s.cs.add(f)
			if err != nil {
				s.errs.add(f.srcPath, err)
			}
		}
	}

	for i := uint(0); i < s.cfg.Jobs; i++ {
		s.wg.Add(1)
		go reader()
	}

	s.contentCh = ch
}

func (s *site) stopContentReader() {
	if s.contentCh == nil {
		return
	}

	close(s.contentCh)
	s.contentCh = nil
	s.wg.Wait()
}

func (s *site) loadLayouts() {
	for w, src := range defaultLayouts {
		s.l[w] = &layout{
			s:       s,
			content: src,
		}
	}

	if s.cfg.Theme != "" {
		themeDir := filepath.Join(s.cfg.ThemesDir, s.cfg.Theme)
		if !dExists(themeDir) {
			s.errs.add(themeDir, errors.New("theme does not exist"))
			return
		}

		s.loadLayoutDir(themeDir, true, 2)
	}

	s.loadLayoutDir(s.cfg.LayoutsDir, false, 1)

	set := p2.NewSet("toner")
	set.Resolver = s

	wg := sync.WaitGroup{}
	tplCh := make(chan *layout, s.cfg.Jobs*2)

	compiler := func() {
		defer wg.Done()

		for lo := range tplCh {
			var err error

			if lo.filePath != "" {
				lo.tpl, err = set.FromFile(lo.filePath)
			} else {
				lo.tpl, err = set.FromString(lo.content)
				if err != nil {
					panic(fmt.Errorf("failed to load default layout %s: %s",
						lo.which,
						err))
				}

				lo.content = ""
			}

			if err != nil {
				s.errs.add(lo.filePath, err)
			}
		}
	}

	for i := uint(0); i < s.cfg.Jobs; i++ {
		wg.Add(1)
		go compiler()
	}

	for _, lo := range s.l {
		tplCh <- lo
	}

	close(tplCh)
	wg.Wait()
}

func (s *site) loadLayoutDir(dir string, isTheme bool, depth int) {
	err := s.walkRoot(dir, func(f file) error {
		if filepath.Ext(f.srcPath) != ".html" {
			pubDir := themePubDir
			if !isTheme {
				pubDir = layoutPubDir
			}

			p := fDropFirst(fDropRoot(s.cfg.Root, f.srcPath))
			f.dstPath = filepath.Join(pubDir, p)

			s.addContent(f, false)
			return nil
		}

		path := fDropRoot(s.cfg.Root, f.srcPath)
		for depth > 0 {
			path = fDropFirst(path)
			depth--
		}

		which := fChangeExt(path, "")
		s.l[which] = &layout{
			s:        s,
			which:    which,
			filePath: f.srcPath,
		}

		return nil
	})

	if err != nil {
		s.errs.add(dir, err)
	}
}

func (s *site) loadContent() {
	err := s.walkRoot(s.cfg.ContentDir, func(f file) error {
		if f.isMeta() {
			return nil
		}

		path := fDropRoot(s.cfg.Root, f.srcPath)
		path = fDropFirst(path)
		f.dstPath = path
		s.addContent(f, true)

		return nil
	})

	if err != nil {
		s.errs.add(s.cfg.ContentDir, err)
	}
}

func (s *site) loadData() {
	return
}

func (s *site) generate() {
	wg := sync.WaitGroup{}

	ch := make(chan contentGenPage, s.cfg.Jobs*2)

	generator := func() {
		defer wg.Done()

		for gp := range ch {
			_, err := gp.generatePage()
			if err != nil {
				s.errs.add(gp.c.f.srcPath, err)
			}
		}
	}

	for i := uint(0); i < s.cfg.Jobs; i++ {
		wg.Add(1)
		go generator()
	}

	for _, c := range s.cs.srcs {
		gp, ok := c.gen.(contentGenPage)
		if ok {
			ch <- gp
		}
	}

	close(ch)
	wg.Wait()
}

func (s *site) walkRoot(p string, cb func(file) error) error {
	p = filepath.Join(s.cfg.Root, p)

	if !dExists(p) {
		return nil
	}

	var walk func(string, func(file) error) error
	walk = func(p string, cb func(file) error) error {
		f, err := os.Open(p)
		if err != nil {
			return err
		}

		infos, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			return err
		}

		for _, info := range infos {
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

func (s *site) isLayout(path string) bool {
	_, f := filepath.Split(path)
	_, ok := defaultLayouts[f]
	return ok
}

func (s *site) findLayout(cpath, which string) *layout {
	for cpath != "." {
		lo := s.l[filepath.Join(cpath, which)]
		if lo != nil {
			return lo
		}

		cpath = filepath.Dir(cpath)
	}

	lo := s.l[which]
	if lo != nil {
		return lo
	}

	// If the template doesn't exist, that's programmer error
	panic(fmt.Errorf("unknown template requested: %s", filepath.Join(cpath, which)))
}

func (s *site) findContent(currFile, path string) (*content, error) {
	return s.cs.find(currFile, path)
}

func (s *site) fCreate(path string) (*os.File, error) {
	return fCreate(path, createFlags, s.cfg.FileMode)
}

func (s *site) fWrite(path string, c []byte) error {
	return fWrite(path, c, s.cfg.FileMode)
}

func (s *site) Resolve(tpl *p2.Template, path string) string {
	if s.isLayout(path) {
		return s.findLayout(filepath.Split(path)).filePath
	}

	if filepath.IsAbs(path) {
		return path
	}

	if tpl != nil {
		abs := tpl.Path()
		if abs != "" {
			base := filepath.Dir(abs)
			return filepath.Join(base, path)
		}
	}

	return path
}
