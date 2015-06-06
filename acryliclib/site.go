package acryliclib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	p2 "github.com/flosch/pongo2"
	"github.com/tdewolff/minify"
	minhtml "github.com/tdewolff/minify/html"
)

type site struct {
	cfg    *Config
	min    *minify.Minify
	mtx    sync.Mutex
	cs     contents
	d      data
	l      map[string]*layout
	assets assets

	stage     buildStage
	wg        sync.WaitGroup
	contentCh chan file

	stats BuildStats
	errs  errs
}

type data map[string]interface{}

type buildStage int

const (
	// Rendering the content
	buildContent buildStage = iota

	// Generating files
	buildFiles
)

var (
	reservedPaths = []string{
		layoutPubDir,
		themePubDir,
	}
	reservedInternalPaths = []string{
		imgPubDir,
		staticPubDir,
	}

	allReservedPaths = []string{}
)

func init() {
	allReservedPaths = append(allReservedPaths, reservedPaths...)
	allReservedPaths = append(allReservedPaths, reservedInternalPaths...)
}

// TODO(astone): menus
// TODO(astone): sitemap.xml
// TODO(astone): rss feeds
// TODO(astone): code highlighting

func newSite(cfg *Config) *site {
	s := &site{
		cfg: cfg,
		min: minify.New(),
		l:   map[string]*layout{},
	}

	s.cs.init(s)
	s.assets.init(s)
	s.min.AddFunc("text/html", minhtml.Minify)

	return s
}

func (s *site) build() (BuildStats, Errors) {
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

	// TODO(astone): check for orphaned content

	if !s.errs.has() {
		s.assets.crunch()
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

	if res := fPathCheckFor(f.dstPath, reservedInternalPaths...); res != "" {
		s.errs.add(f.srcPath, fmt.Errorf("use of reserved path `%s` is not allowed", res))
		return
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
		themeDir := filepath.Join(s.cfg.Root, s.cfg.ThemesDir, s.cfg.Theme)
		if !dExists(themeDir) {
			s.errs.add(themeDir, errors.New("theme does not exist"))
			return
		}

		themeDir = filepath.Join(s.cfg.ThemesDir, s.cfg.Theme)
		s.loadLayoutDir(themeDir, true, 2)
	}

	s.loadLayoutDir(s.cfg.LayoutsDir, false, 1)

	set := p2.NewSet("acrylic")
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
	dstPath := func(src string) string {
		pubDir := themePubDir
		if !isTheme {
			pubDir = layoutPubDir
		}

		p := fDropFirst(fDropRoot(s.cfg.Root, src))
		return filepath.Join(pubDir, p)
	}

	err := s.walkRoot(dir,
		func(f file) error {
			if filepath.Ext(f.srcPath) != ".html" {
				f.dstPath = dstPath(f.srcPath)
				s.addContent(f, false)
				return nil
			}

			path := fDropRoot(s.cfg.Root, f.srcPath)
			currDepth := depth
			for currDepth > 0 {
				path = fDropFirst(path)
				currDepth--
			}

			which := fChangeExt(path, "")
			s.l[which] = &layout{
				s:        s,
				which:    which,
				filePath: f.srcPath,
			}

			return nil
		},
		func(dir string, indexes []file) error {
			return s.checkDirIndexes(false, dir, dstPath, indexes)
		})

	if err != nil {
		s.errs.add(dir, err)
	}
}

func (s *site) loadContent() {
	dstPath := func(src string) string {
		path := fDropRoot(s.cfg.Root, src)
		return fDropFirst(path)
	}

	err := s.walkRoot(s.cfg.ContentDir,
		func(f file) error {
			f.dstPath = dstPath(f.srcPath)
			s.addContent(f, true)
			return nil
		},
		func(dir string, indexes []file) error {
			return s.checkDirIndexes(true, dir, dstPath, indexes)
		})

	if err != nil {
		s.errs.add(s.cfg.ContentDir, err)
	}
}

func (s *site) checkDirIndexes(
	isContent bool,
	dir string,
	dstPath func(string) string,
	indexes []file) error {

	if len(indexes) == 0 {
		indexes = append(indexes, file{
			srcPath:    filepath.Join(dir, "index.html"),
			isImplicit: true,
		})
	}

	root := filepath.Join(s.cfg.Root, s.cfg.ContentDir)

	for _, f := range indexes {
		f.layoutName = "_list"

		if dir == root {
			f.layoutName = "_index"
		}

		f.dstPath = dstPath(f.srcPath)
		s.addContent(f, isContent)

	}
	return nil
}

func (s *site) loadData() {
	// TODO(astone): load data
	return
}

func (s *site) generate() {
	wg := sync.WaitGroup{}

	ch := make(chan *content, s.cfg.Jobs*2)

	generator := func() {
		defer wg.Done()

		for c := range ch {
			// Don't generate layout and theme pages unless explicitly requested
			if res := fPathCheckFor(c.f.dstPath, allReservedPaths...); res != "" {
				continue
			}

			c.gen.generatePage()
		}
	}

	for i := uint(0); i < s.cfg.Jobs; i++ {
		wg.Add(1)
		go generator()
	}

	for _, c := range s.cs.srcs {
		if c.gen.is(contPage) {
			ch <- c
		}
	}

	close(ch)
	wg.Wait()
}

func (s *site) walkRoot(
	p string,
	fCb func(file) error,
	indexesCb func(string, []file) error) error {

	p = filepath.Join(s.cfg.Root, p)

	if !dExists(p) {
		return nil
	}

	var walk func(string) error
	walk = func(p string) error {
		f, err := os.Open(p)
		if err != nil {
			return err
		}

		infos, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			return err
		}

		var indexes []file

		for _, info := range infos {
			p := filepath.Join(p, info.Name())

			if info.IsDir() {
				err = walk(p)
			} else {
				f := file{
					srcPath: p,
				}

				if f.isMeta() {
					continue
				}

				if f.isIndex() {
					indexes = append(indexes, f)
					continue
				}

				err = fCb(f)
			}

			if err != nil {
				return err
			}
		}

		return indexesCb(p, indexes)
	}

	return walk(p)
}

func (s *site) isLayout(path string) bool {
	_, f := filepath.Split(path)
	_, ok := defaultLayouts[f]
	return ok
}

func (s *site) findLayout(cpath, which string, failIfNotFound bool) *layout {
	for cpath != "." {
		lo := s.l[filepath.Join(cpath, which)]
		if lo != nil {
			return lo
		}

		cpath = filepath.Dir(cpath)
	}

	if strings.HasPrefix(which, "/") {
		which = which[1:]
	}

	lo := s.l[which]
	if lo != nil {
		return lo
	}

	if !failIfNotFound {
		return nil
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
		dir, which := filepath.Split(path)
		return s.findLayout(dir, which, true).filePath
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
