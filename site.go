package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flosch/pongo2"
	"github.com/tdewolff/minify"
	min_css "github.com/tdewolff/minify/css"
	min_html "github.com/tdewolff/minify/html"
	min_js "github.com/tdewolff/minify/js"
	"gopkg.in/yaml.v2"
)

type site struct {
	args    []string
	cfg     *config
	logOut  io.Writer
	baseDir string
	errs    errs

	workWg sync.WaitGroup
	workCh chan func()

	ss *siteState
}

type siteState struct {
	min       *minify.Minify
	tmplSet   *pongo2.TemplateSet
	buildTime time.Time

	mtx     sync.Mutex
	data    map[string]interface{}
	pages   pages
	imgs    images
	blobs   []*blob
	js      []string
	css     []string
	publics map[string]struct{}
}

type pages struct {
	byCat map[string][]*page
}

type page struct {
	src        string
	dst        string
	sortName   string
	Cat        string
	Title      string
	Date       time.Time
	Content    string
	Summary    string
	URL        string
	isListPage bool
	Meta       map[string]interface{}
}

type pageSlice []*page
type imageSlice []*image

type images struct {
	all   []*image
	imgs  map[string]*image
	byCat map[string][]*image
}

type image struct {
	s         *site
	src       string
	dst       string
	sortName  string
	info      os.FileInfo
	Cat       string
	Title     string
	Date      time.Time
	url       string
	Meta      map[string]interface{}
	inGallery bool
}

type blob struct {
	src string
	dst string
}

const (
	scissors     = `<!-- >8 acrylic-content -->`
	scissorsEnd  = `<!-- acrylic-content 8< -->`
	moreScissors = `<!--more-->`
)

var (
	frontMatterStart = []byte("---\n")
	frontMatterEnd   = []byte("\n---\n")

	reCSSURL    = regexp.MustCompile(`url\("?(.*?)"?\)`)
	reCSSScaled = regexp.MustCompile(`.*(\.((\d*)x(\d*)(c?)(-q(\d*))?)).*`)
)

func (s *site) build() (ok bool) {
	s.ss = newSiteState(s)
	defer func() {
		s.ss = nil
	}()

	s.withPool(func() {
		s.walk(s.cfg.DataDir, s.loadData)
		s.walk(s.cfg.ContentDir, s.loadContent)
		s.walk(s.cfg.AssetsDir, s.loadAssetImages)
		s.walk(s.cfg.PublicDir, s.loadPublic)
	})

	if !s.checkErrs() {
		return
	}

	s.ss.loadFinished()

	s.withPool(func() {
		s.renderPages()
		s.renderAssets()
		s.copyBlobs()
	})

	if !s.checkErrs() {
		return
	}

	s.withPool(func() {
		s.renderListPages()
	})

	if !s.checkErrs() {
		return
	}

	paths := []string{}
	for path := range s.ss.publics {
		paths = append(paths, path)
	}

	// Sorted in reverse, this should make sure that any empty directories are
	// removed recursively
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		os.Remove(path)
	}

	return true
}

func (s *site) withPool(fn func()) {
	wg := sync.WaitGroup{}
	s.workCh = make(chan func(), 8192)

	for i := 0; i < runtime.GOMAXPROCS(-1); i++ {
		wg.Add(1)
		go s.jobRunner(&wg)
	}

	fn()

	s.workWg.Wait()
	close(s.workCh)
	wg.Wait()
}

func (s *site) jobRunner(exitWg *sync.WaitGroup) {
	defer exitWg.Done()

	for fn := range s.workCh {
		fn()
		s.workWg.Done()
	}
}

func (s *site) workIt(fn func()) {
	s.workWg.Add(1)

	select {
	case s.workCh <- fn:
	default:
		go func() {
			s.workCh <- fn
		}()
	}
}

func (s *site) renderPages() {
	for _, pgs := range s.ss.pages.byCat {
		for _, pg := range pgs {
			if pg.isListPage {
				continue
			}

			func(pg *page) {
				s.workIt(func() {
					s.renderPage(pg)
				})
			}(pg)
		}
	}
}

func (s *site) copyFile(src, dst string) {
	err := fCopy(src, dst)
	if err != nil {
		s.errs.add(src, fmt.Errorf("failed to copy: %v", err))
		return
	}

	info, err := os.Stat(src)
	if err != nil {
		s.errs.add(src, err)
		return
	}

	os.Chtimes(dst, info.ModTime(), info.ModTime())
	s.ss.markUsed(dst)
}

func (s *site) renderAssets() {
	assetPath := func(path string) (src, dst string) {
		src = filepath.Join(s.baseDir, s.cfg.AssetsDir, path)
		dst = filepath.Join(s.baseDir, s.cfg.PublicDir, s.cfg.AssetsDir, path)
		return
	}

	rawCopy := func(src string) {
		s.copyFile(assetPath(src))
	}

	if s.cfg.Debug {
		for _, js := range s.cfg.JS {
			func(js string) {
				s.workIt(func() {
					rawCopy(js)
				})
			}(js)
		}

		for _, css := range s.cfg.CSS {
			func(css string) {
				s.workIt(func() {
					src, dst := assetPath(css)

					if filepath.Ext(css) == ".scss" {
						s.compileScssToFile(src, dst)
					} else {
						rawCopy(css)
					}

					s.processCSSAssets(fChangeExt(dst, ".css"))
				})
			}(css)
		}

		return
	}

	s.workIt(func() {
		b := bytes.Buffer{}
		for _, js := range s.cfg.JS {
			src, _ := assetPath(js)

			f, err := os.Open(src)
			if err != nil {
				s.errs.add(src, err)
				return
			}

			_, err = b.ReadFrom(f)
			f.Close()

			if err != nil {
				s.errs.add(src, err)
				return
			}
		}

		_, dst := assetPath("all.js")
		f, err := fCreate(dst)
		if err != nil {
			s.errs.add(dst, err)
			return
		}

		defer f.Close()

		if len(s.cfg.JSCompiler) > 0 {
			cmd := exec.Command(s.cfg.JSCompiler[0], s.cfg.JSCompiler[1:]...)
			cmd.Stdin = &b
			cmd.Stdout = f

			eb := bytes.Buffer{}
			cmd.Stderr = &eb

			err = cmd.Run()
			if err != nil {
				err = fmt.Errorf("%v: %v", err, eb.String())
			}
		} else {
			err = s.ss.min.Minify("text/javascript", f, &b)
		}

		if err != nil {
			s.errs.add(dst, err)
		} else {
			s.ss.markUsed(dst)
		}
	})

	s.workIt(func() {
		b := bytes.Buffer{}
		for _, css := range s.cfg.CSS {
			src, _ := assetPath(css)

			var err error
			if filepath.Ext(src) == ".scss" {
				err = s.compileScss(src, &b)
			} else {
				f, err := os.Open(src)
				if err != nil {
					s.errs.add(src, err)
					return
				}

				_, err = b.ReadFrom(f)
				f.Close()
			}

			if err != nil {
				s.errs.add(src, err)
				return
			}
		}

		_, dst := assetPath("all.css")
		f, err := fCreate(dst)
		if err != nil {
			s.errs.add(dst, err)
			return
		}

		err = s.ss.min.Minify("text/css", f, &b)
		f.Close()

		if err != nil {
			s.errs.add(dst, err)
		} else {
			s.ss.markUsed(dst)
			s.processCSSAssets(dst)
		}
	})
}

func (s *site) compileScss(src string, out io.Writer) error {
	args := append([]string{}, s.cfg.SassCompiler...)
	args = append(args, src)

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = out

	eb := bytes.Buffer{}
	cmd.Stderr = &eb

	err := cmd.Run()
	if err != nil || eb.Len() > 0 {
		return fmt.Errorf("execute failed: %v, stderr=%s", err, eb.String())
	}

	return nil
}

func (s *site) compileScssToFile(src, dstPath string) {
	var dst io.ReadWriteCloser

	dstPath = fChangeExt(dstPath, ".css")

	dst, err := fCreate(dstPath)
	if err == nil {
		defer dst.Close()
		err = s.compileScss(src, dst)
	}

	if err != nil {
		s.errs.add(src, err)
	} else {
		s.ss.markUsed(dstPath)
	}
}

func (s *site) processCSSAssets(path string) {
	sheet, err := ioutil.ReadFile(path)
	if err != nil {
		s.errs.add(path, err)
		return
	}

	pfx := filepath.Clean("/" + s.cfg.AssetsDir + "/")

	matches := reCSSURL.FindAllSubmatch(sheet, -1)
	for _, match := range matches {
		absURL := string(match[1])

		if !strings.HasPrefix(absURL, pfx) {
			continue
		}

		url := absURL
		w := 0
		h := 0
		crop := false
		quality := 100

		parts := reCSSScaled.FindStringSubmatch(url)
		if len(parts) > 0 {
			w, _ = strconv.Atoi(parts[3])
			h, _ = strconv.Atoi(parts[4])
			crop = len(parts[5]) > 0
			quality, _ = strconv.Atoi(parts[7])

			url = strings.Replace(url, parts[1], "", -1)
		}

		path := filepath.Join(s.baseDir, url)

		img := s.ss.imgs.get(path)
		if img == nil {
			s.errs.add(path,
				fmt.Errorf("image not found: %s (resolved to %s)", url, path))
			return
		}

		final := img.Scale(w, h, crop, quality)
		sheet = bytes.Replace(
			sheet,
			match[0],
			[]byte(fmt.Sprintf(`url("%s")`, final)),
			-1)
	}

	err = ioutil.WriteFile(path, sheet, 0640)
	if err != nil {
		s.errs.add(path, err)
		return
	}
}

func (s *site) copyBlobs() {
	for _, b := range s.ss.blobs {
		func(b *blob) {
			s.workIt(func() {
				s.copyFile(b.src, b.dst)
			})
		}(b)
	}
}

func (s *site) renderListPages() {
	for _, pgs := range s.ss.pages.byCat {
		for _, pg := range pgs {
			if !pg.isListPage {
				continue
			}

			func(pg *page) {
				s.workIt(func() {
					s.renderListPage(pg)
				})
			}(pg)
		}
	}
}

func (s *site) renderPage(pg *page) {
	tmpl, err := s.ss.tmplSet.FromString(pg.Content)
	if err != nil {
		s.errs.add(pg.src, err)
		return
	}

	content, err := tmpl.Execute(s.tmplVars(pg))
	if err != nil {
		s.errs.add(pg.src, err)
		return
	}

	start := strings.Index(content, scissors)
	end := strings.Index(content, scissorsEnd)

	pg.Content = ""
	if start >= 0 && end >= 0 {
		pg.Content = content[start+len(scissors) : end]

		end := strings.Index(pg.Content, moreScissors)
		if end >= 0 {
			pg.Summary = pg.Content[:end]
		}
	}

	s.writePage(pg, pg.dst, content)
}

func (s *site) renderListPage(pg *page) {
	tmpl, err := s.ss.tmplSet.FromString(pg.Content)
	if err != nil {
		s.errs.add(pg.src, err)
		return
	}

	pages := s.ss.pages.posts(pg.Cat)

	total := len(pages)
	pageCount := int(math.Ceil(float64(total) / float64(s.cfg.PerPage)))

	for i := 0; i < pageCount; i++ {
		listStart := i * s.cfg.PerPage
		listEnd := listStart + s.cfg.PerPage

		if listEnd > total {
			listEnd = total
		}

		content, err := tmpl.Execute(s.tmplVars(pg).Update(
			pongo2.Context{
				"page":        pg,
				"pages":       pages[listStart:listEnd],
				"pageNum":     i + 1,
				"listHasNext": i < (pageCount - 1),
			}))
		if err != nil {
			s.errs.add(pg.src, err)
			return
		}

		dst := pg.dst
		if i > 0 {
			dst = filepath.Join(
				s.baseDir,
				s.cfg.PublicDir,
				filepath.Dir(pg.URL),
				"page", fmt.Sprintf("%d", i+1),
				"index.html")
		}

		s.writePage(pg, dst, content)
	}
}

func (s *site) writePage(pg *page, dst, content string) {
	s.ss.markUsed(dst)

	if !s.cfg.Debug {
		b := bytes.Buffer{}
		err := s.ss.min.Minify("text/html", &b, strings.NewReader(content))
		if err != nil {
			s.errs.add(pg.src, err)
			return
		}

		content = b.String()
	}

	err := fWrite(dst, []byte(content))
	if err != nil {
		s.errs.add(pg.src, err)
		return
	}
}

func (s *site) tmplVars(pg *page) pongo2.Context {
	return pongo2.Context{
		"ac":   newTmplAC(s, pg),
		"page": pg,
	}
}

func (s *site) checkErrs() bool {
	errs := s.errs.String()

	if len(errs) == 0 {
		return true
	}

	fmt.Fprintf(s.logOut, errs)

	return false
}

func (s *site) walk(dir string, cb func(string, os.FileInfo)) {
	dir = filepath.Join(s.baseDir, dir)

	if !dExists(dir) {
		return
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.errs.add(path, err)
			return nil
		}

		s.workIt(func() {
			cb(path, info)
		})

		return nil
	})
}

func (s *site) loadData(file string, info os.FileInfo) {
	if info.IsDir() {
		return
	}

	var data []byte
	var err error

	name := fDropRoot(s.baseDir, s.cfg.DataDir, file)
	cached := filepath.Join(s.baseDir, s.cfg.CacheDir, "data", name)

	if fExists(cached) && !fSrcChanged(file, cached) {
		data, err = ioutil.ReadFile(cached)
		if err != nil {
			s.errs.add(file, err)
			return
		}

		until := time.Unix(int64(binary.BigEndian.Uint64(data[:8])), 0)
		if until.Before(time.Now()) {
			data = nil
		} else {
			data = data[8:]
		}
	}

	if len(data) == 0 {
		if (info.Mode() & 0111) != 0 {
			cmd := exec.Command(file)

			ob := bytes.Buffer{}
			cmd.Stdout = &ob

			eb := bytes.Buffer{}
			cmd.Stderr = &eb

			err := cmd.Run()
			if err != nil || eb.Len() > 0 {
				s.errs.add(file, fmt.Errorf("execute failed: %v: %s",
					err,
					eb.String()))
				return
			}

			data = ob.Bytes()
		} else {
			data, err = ioutil.ReadFile(file)
			if err != nil {
				s.errs.add(file, err)
				return
			}
		}
	}

	var v interface{}
	err = json.Unmarshal(data, &v)
	if err != nil {
		s.errs.add(file, err)
		return
	}

	if v, ok := v.(map[string]interface{}); ok {
		if until, ok := v["acrylic_expires"].(float64); ok {
			b := bytes.Buffer{}
			binary.Write(&b, binary.BigEndian, uint64(until))
			b.Write(data)

			err = fWrite(cached, b.Bytes())
			if err != nil {
				s.errs.add(file, fmt.Errorf("failed to write cache file: %v", err))
				return
			}

			os.Chtimes(cached, info.ModTime(), info.ModTime())
		}

		delete(v, "acrylic_expires")
	}

	s.ss.mtx.Lock()
	s.ss.data[name] = v
	s.ss.mtx.Unlock()
}

func (s *site) loadContent(file string, info os.FileInfo) {
	if info.IsDir() {
		return
	}

	switch filepath.Ext(file) {
	case ".html":
		s.loadPage(file, info)

	case ".jpg", ".gif", ".png", ".svg":
		s.loadImg(file, info, true)

	case ".meta":
		// Ignore these

	default:
		s.loadBlob(file, info)
	}
}

func (s *site) loadAssetImages(file string, info os.FileInfo) {
	if !info.IsDir() {
		switch filepath.Ext(file) {
		case ".jpg", ".gif", ".png", ".svg":
			s.loadImg(file, info, false)
		}
	}
}

func (s *site) loadPage(file string, info os.FileInfo) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		s.errs.add(file, err)
		return
	}

	fm := map[string]interface{}{}
	if bytes.HasPrefix(c, frontMatterStart) {
		end := bytes.Index(c, frontMatterEnd)
		if end == -1 {
			s.errs.add(file, fmt.Errorf("missing front matter end"))
			return
		}

		fmb := c[len(frontMatterStart):end]
		err = yaml.Unmarshal(fmb, &fm)
		if err != nil {
			s.errs.add(file, err)
			return
		}

		c = c[end+len(frontMatterEnd):]
	}

	category, date, name, sortName, url, dst := s.getOutInfo(file, s.cfg.ContentDir, true)

	title := ""
	if t, ok := fm["title"].(string); ok {
		title = t
	} else {
		title = sToTitle(name)
	}

	isListPage := false
	if b, ok := fm["list_page"].(bool); ok {
		isListPage = b
	}

	s.ss.addPage(&page{
		src:        file,
		dst:        dst,
		sortName:   sortName,
		Cat:        category,
		Title:      title,
		Date:       date,
		Content:    string(c),
		URL:        url,
		isListPage: isListPage,
		Meta:       fm,
	})
}

func (s *site) loadImg(file string, info os.FileInfo, isContent bool) {
	rootDir := ""
	if isContent {
		rootDir = s.cfg.ContentDir
	}

	category, date, _, sortName, url, dst := s.getOutInfo(file, rootDir, false)

	metaFile := file + ".meta"
	fm := map[string]interface{}{}
	if fExists(metaFile) {
		b, err := ioutil.ReadFile(metaFile)
		if err != nil {
			s.errs.add(file, err)
			return
		}

		err = yaml.Unmarshal(b, &fm)
		if err != nil {
			s.errs.add(file, err)
			return
		}
	} else if strings.HasPrefix(file, s.cfg.ContentDir) {
		fWrite(metaFile, []byte("---\ntitle: \n---\n"))
	}

	title := ""
	if t, ok := fm["title"].(string); ok {
		title = t
	}

	inGallery := isContent
	if g, ok := fm["gallery"].(bool); ok {
		inGallery = g
	}

	s.ss.addImage(&image{
		s:         s,
		src:       file,
		dst:       dst,
		sortName:  sortName,
		info:      info,
		Cat:       category,
		Title:     title,
		Date:      date,
		url:       url,
		Meta:      fm,
		inGallery: inGallery,
	})
}

func (s *site) loadBlob(file string, info os.FileInfo) {
	_, _, _, _, _, dst := s.getOutInfo(file, s.cfg.ContentDir, false)
	s.ss.addBlob(file, dst)
}

func (s *site) loadPublic(file string, info os.FileInfo) {
	s.ss.addPublic(file)
}

func (s *site) getOutInfo(file, dir string, isPage bool) (
	cat string,
	date time.Time,
	name, sortName, url, dst string) {

	name = fDropRoot(s.baseDir, dir, file)

	if strings.Count(name, "/") == 0 {
		url = "/" + filepath.Clean(name)
		dst = filepath.Join(s.baseDir, s.cfg.PublicDir, name)
		return
	}

	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		cat = parts[0]
		sortName = parts[1]
		date, name = s.parseName(parts[1])
	} else {
		last := parts[len(parts)-2]
		_, ok := sToDate(last)
		if ok {
			date, name = s.parseName(last)
			cat = filepath.Join(parts[0 : len(parts)-2]...)
		} else {
			last = parts[len(parts)-1]
			date, name = s.parseName(last)
			cat = filepath.Join(parts[0 : len(parts)-1]...)
		}

		sortName = last
	}

	name = fChangeExt(name, "")
	if date.IsZero() {
		url = cat
	} else {
		url = filepath.Join(cat, date.Format("2006/01/02"), name)
	}

	dst = filepath.Join(s.baseDir, s.cfg.PublicDir, url)

	if isPage {
		dst = filepath.Join(dst, "index.html")
		url += "/"
	} else {
		base := filepath.Base(file)
		dst = filepath.Join(dst, base)
		url = filepath.Join(url, base)
		sortName = filepath.Join(sortName, base)
	}

	url = "/" + url

	return
}

func (s *site) parseName(name string) (time.Time, string) {
	t, ok := sToDate(name)
	if !ok {
		return t, name
	}

	return t, strings.Trim(name[len(sDateFormat):], "-")
}

func newSiteState(s *site) *siteState {
	tmplDir := filepath.Join(s.baseDir, s.cfg.TemplatesDir)

	ss := &siteState{
		min: minify.New(),
		tmplSet: pongo2.NewSet(
			"acrylic",
			pongo2.MustNewLocalFileSystemLoader(tmplDir)),
		buildTime: time.Now(),
		data:      map[string]interface{}{},
		pages: pages{
			byCat: map[string][]*page{},
		},
		imgs: images{
			imgs:  map[string]*image{},
			byCat: map[string][]*image{},
		},
		publics: map[string]struct{}{},
	}

	ss.min.AddFunc("text/css", min_css.Minify)
	ss.min.AddFunc("text/html", min_html.Minify)
	ss.min.AddFunc("text/javascript", min_js.Minify)

	ss.tmplSet.Globals.Update(pongo2.Context{
		"cfg":  s.cfg,
		"data": ss.data,
	})

	return ss
}

func (ss *siteState) markUsed(dst string) {
	ss.mtx.Lock()
	delete(ss.publics, dst)
	ss.mtx.Unlock()
}

func (ss *siteState) loadFinished() {
	ss.pages.sort()
	ss.imgs.sort()
}

func (ss *siteState) addPage(p *page) {
	ss.mtx.Lock()
	ss.pages.add(p)
	ss.mtx.Unlock()
}

func (ss *siteState) addImage(img *image) {
	ss.mtx.Lock()
	ss.imgs.add(img)
	ss.mtx.Unlock()
}

func (ss *siteState) addBlob(src, dst string) {
	ss.mtx.Lock()
	ss.blobs = append(ss.blobs, &blob{
		src: src,
		dst: dst,
	})
	ss.mtx.Unlock()
}

func (ss *siteState) addJS(file string) {
	ss.mtx.Lock()
	ss.js = append(ss.js, file)
	ss.mtx.Unlock()
}

func (ss *siteState) addCSS(file string) {
	ss.mtx.Lock()
	ss.css = append(ss.css, file)
	ss.mtx.Unlock()
}

func (ss *siteState) addPublic(file string) {
	ss.mtx.Lock()
	ss.publics[file] = struct{}{}
	ss.mtx.Unlock()
}

func (pgs *pages) add(p *page) {
	cat := pgs.byCat[p.Cat]
	cat = append(cat, p)
	pgs.byCat[p.Cat] = cat
}

func (pgs *pages) sort() {
	for _, pgs := range pgs.byCat {
		sort.Sort(pageSlice(pgs))
	}
}

func (pgs *pages) posts(cat string) []*page {
	var posts []*page

	add := func(ps []*page) {
		for _, p := range ps {
			if !p.Date.IsZero() {
				posts = append(posts, p)
			}
		}
	}

	if len(cat) == 0 {
		for _, ps := range pgs.byCat {
			add(ps)
		}

		sort.Sort(pageSlice(posts))
	} else {
		add(pgs.byCat[cat])
	}

	return posts
}

func (imgs *images) add(img *image) {
	cat := imgs.byCat[img.Cat]
	cat = append(cat, img)
	imgs.byCat[img.Cat] = cat

	imgs.imgs[img.src] = img

	imgs.all = append(imgs.all, img)
}

func (imgs *images) sort() {
	sort.Sort(imageSlice(imgs.all))

	for _, ims := range imgs.byCat {
		sort.Sort(imageSlice(ims))
	}
}

func (imgs *images) get(path string) *image {
	return imgs.imgs[path]
}

func (img *image) IsGif() bool {
	return img != nil && filepath.Ext(img.src) == ".gif"
}

func (img *image) Scale(w, h int, crop bool, quality int) string {
	if img == nil {
		return "<IMAGE NOT FOUND>"
	}

	if quality == 0 {
		quality = 100
	}

	if w == 0 && h == 0 && !crop && quality == 100 {
		img.s.workIt(img.copy)
		return img.cacheBustedURL(img.url)
	}

	dims := ""

	// Use something like '400x' to scale to a width of 400
	if w != 0 {
		dims += fmt.Sprintf("%d", w)
	}

	dims += "x"

	if h != 0 {
		dims += fmt.Sprintf("%d", h)
	}

	suffix := dims
	scaleDims := dims

	if crop {
		suffix += "c"
		scaleDims += "^"
	}

	if quality != 100 {
		suffix += fmt.Sprintf("-q%d", quality)
	}

	suffix += filepath.Ext(img.dst)

	img.s.workIt(func() {
		dst := fChangeExt(img.dst, suffix)
		if !fSrcChanged(img.src, dst) {
			img.updateDst(dst)
			return
		}

		args := []string{
			img.src,
			"-quality", fmt.Sprintf("%d", quality),
			"-scale", scaleDims,
		}

		if crop {
			args = append(args,
				"-gravity", "center",
				"-extent", dims)
		}

		args = append(args, dst)
		err := fCreateParents(dst)
		if err != nil {
			img.s.errs.add(img.src, fmt.Errorf("failed to scale: %v", err))
			return
		}

		cmd := exec.Command("convert", args...)

		eb := bytes.Buffer{}
		cmd.Stderr = &eb

		err = cmd.Run()
		if err != nil {
			img.s.errs.add(img.src,
				fmt.Errorf("failed to scale: %v: stderr=%s",
					err,
					eb.String()))
			return
		}

		img.updateDst(dst)
	})

	return img.cacheBustedURL(fChangeExt(img.url, suffix))
}

func (img *image) cacheBustedURL(url string) string {
	if !img.s.cfg.CacheBust {
		return url
	}

	return fmt.Sprintf("%s?%d", url, img.info.ModTime().Unix())
}

func (img *image) copy() {
	err := fCopy(img.src, img.dst)
	if err != nil {
		img.s.errs.add(img.src, fmt.Errorf("while copying: %v", err))
	} else {
		img.updateDst(img.dst)
	}
}

func (img *image) updateDst(dst string) {
	os.Chtimes(dst, img.info.ModTime(), img.info.ModTime())
	img.s.ss.markUsed(dst)
}

func (ps pageSlice) Len() int           { return len(ps) }
func (ps pageSlice) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }
func (ps pageSlice) Less(i, j int) bool { return ps[i].sortName > ps[j].sortName }

func (is imageSlice) Len() int           { return len(is) }
func (is imageSlice) Swap(i, j int)      { is[i], is[j] = is[j], is[i] }
func (is imageSlice) Less(i, j int) bool { return is[i].sortName > is[j].sortName }
