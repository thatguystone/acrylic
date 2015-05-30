package toner

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	p2 "github.com/flosch/pongo2"
)

type layouts struct {
	s    *site
	tpls map[string]*layout
}

const (
	assetsKey  = "__tonerAssets__"
	relPathKey = "__tonerRelPath__"
)

func (l *layouts) init() error {
	l.tpls = map[string]*layout{}
	tmps := map[string]layout{}

	for w, src := range defaultLayouts {
		tmps[w] = layout{
			content: src,
		}
	}

	themeDir := filepath.Join(l.s.cfg.ThemesDir, l.s.cfg.Theme)
	if l.s.cfg.Theme != "" && !l.s.dExists(themeDir) {
		return fmt.Errorf("theme %s doesn't exist", themeDir)
	}

	err := l.loadDir(tmps, themeDir, 2)
	if err != nil {
		return fmt.Errorf("in theme %s: %s", l.s.cfg.Theme, err)
	}

	err = l.loadDir(tmps, l.s.cfg.LayoutsDir, 1)
	if err != nil {
		return err
	}

	set := p2.NewSet("toner")
	set.Resolver = l

	type ctpl struct {
		which string
		lout  layout
	}

	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	tplCh := make(chan ctpl, l.s.cfg.Jobs*2)
	errCh := make(chan []string, l.s.cfg.Jobs)

	compiler := func() {
		defer wg.Done()

		errs := []string{}
		for c := range tplCh {
			var tpl *p2.Template
			var err error

			if c.lout.path != "" {
				of, err := l.s.fs.Open(c.lout.path)
				if err == nil {
					b, err := ioutil.ReadAll(of)
					of.Close()

					if err == nil {
						tpl, err = set.FromString(string(b))
					}
				}
			} else {
				tpl, err = set.FromString(c.lout.content)
				if err != nil {
					panic(fmt.Errorf("failed to load default layout %s: %s",
						c.which,
						err))
				}
			}

			lo := &layout{
				tpl:    tpl,
				path:   c.lout.path,
				assets: tplAssets{assets: &l.s.a},
			}

			if err == nil {
				err = lo.prerender(l.s)
			}

			if err == nil {
				mtx.Lock()
				l.tpls[c.which] = lo
				mtx.Unlock()
			}

			if err != nil {
				errs = append(errs, fmt.Sprintf("in %s: %s", c.lout.path, err))
			}
		}

		errCh <- errs
	}

	for i := uint(0); i < l.s.cfg.Jobs; i++ {
		wg.Add(1)
		go compiler()
	}

	for w, lout := range tmps {
		tplCh <- ctpl{
			which: w,
			lout:  lout,
		}
	}

	close(tplCh)
	wg.Wait()
	close(errCh)

	errs := []string{}
	for e := range errCh {
		errs = append(errs, e...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to compile layouts: \n\t%s",
			strings.Join(errs, "\n\t"))
	}

	return nil
}

func (l *layouts) loadDir(tmps map[string]layout, dir string, depth int) error {
	return l.s.walkRoot(dir, func(f file) error {
		if filepath.Ext(f.srcPath) != ".html" {
			return nil
		}

		path := f.srcPath
		for depth > 0 {
			path = fDropFirst(path)
			depth--
		}

		which := fChangeExt(path, "")
		tmps[which] = layout{
			path: f.srcPath,
		}

		return nil
	})
}

func (l *layouts) find(path, which string) *layout {
	for len(path) > 0 {
		lo := l.tpls[filepath.Join(path, which)]
		if lo != nil {
			return lo
		}

		path = filepath.Dir(path)
	}

	lo := l.tpls[which]
	if lo != nil {
		return lo
	}

	// If the template doesn't exist, that's programmer error
	panic(fmt.Errorf("unknown template requested: %s", filepath.Join(path, which)))
}

func (l *layouts) isLayout(path string) bool {
	for _, n := range layoutNames {
		if strings.HasSuffix(path, n) {
			return true
		}
	}

	return false
}

func (l *layouts) Resolve(tpl *p2.Template, path string) string {
	if l.isLayout(path) {
		return l.find(filepath.Split(path)).path
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
