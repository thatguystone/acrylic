package acrylib

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	p2 "github.com/flosch/pongo2"
)

type contents struct {
	s           *site
	mtx         sync.Mutex
	srcs        map[string]*content // All available content: srcPath -> content
	dsts        map[string]*content // All rendered content: dstPath -> content
	dirHasIndex map[string]*bool    // Which directories have index pages
}

type content struct {
	cs       *contents
	f        file
	cpath    string
	metaEnd  int
	meta     *meta
	tplCont  TplContent
	gen      contentGenWrapper
	assetOrd assetOrdering
	deets    contentDetails
}

type contentDetails struct {
	mtx            sync.Mutex
	analyzed       bool
	summary        string
	wordCount      int
	fuzzyWordCount int
}

const (
	layoutPubDir = "layout"
	themePubDir  = "theme"
)

var (
	metaDelim         = []byte("---")
	metaHeaderTypeLen = len(metaDelim) + 1 + 4

	bannedContentTags = []string{
		"css_all",
		"extends",
		"js_all",
	}

	bannedContentFilters = []string{}

	// This can be shared: it has no globals set, and it doesn't lock or
	// anything.
	p2ContentSet = p2.NewSet("content")
)

func (cs *contents) init(s *site) {
	cs.s = s
	cs.srcs = map[string]*content{}
	cs.dsts = map[string]*content{}
	cs.dirHasIndex = map[string]*bool{}
}

func (cs *contents) add(f file) error {
	if f.layoutName == "" {
		f.layoutName = "_single"
	}

	s := cs.s
	cpath := fChangeExt(f.dstPath, "")

	c := &content{
		cs:    cs,
		f:     f,
		cpath: cpath,
		meta:  &meta{},
	}

	c.gen = getContentGener(s, c, filepath.Ext(c.f.srcPath))

	publicDst := filepath.Join(s.cfg.Root, s.cfg.PublicDir)
	if s.cfg.UglyURLs || !c.gen.is(contPage) || c.f.isIndex() {
		if c.cpath == "index" {
			c.cpath = "<index>"
		} else if c.f.isIndex() {
			c.cpath = filepath.Dir(c.cpath)
		}

		c.f.dstPath = filepath.Join(publicDst, c.f.dstPath)
	} else {
		c.f.dstPath = filepath.Join(publicDst, c.cpath, "index.html")
	}

	c.f.dstPath = fChangeExt(c.f.dstPath, c.gen.gener.finalExt(c))

	err := c.load()
	if err != nil {
		return err
	}

	c.tplCont.init(cs.s, c)
	if !c.shouldPublish() {
		return nil
	}

	cs.mtx.Lock()

	dstDir := filepath.Dir(c.f.dstPath)
	currDir := dstDir

	for strings.HasPrefix(currDir, publicDst) {
		has, ok := cs.dirHasIndex[currDir]
		if !ok {
			has = new(bool)
			cs.dirHasIndex[currDir] = has
		}

		currDir = filepath.Dir(currDir)
	}

	has := cs.dirHasIndex[dstDir]
	*has = *has || (c.gen.is(contPage) && c.f.isIndex())
	cs.srcs[c.f.srcPath] = c

	cs.mtx.Unlock()

	err = cs.s.tplSite.addContent(&c.tplCont)

	if err == nil && c.f.layoutName == "_list" && c.meta.rss() {
		cs.add(file{
			srcPath:    filepath.Join(filepath.Dir(c.f.srcPath), "feed.rss"),
			dstPath:    filepath.Join(c.cpath, "feed.rss"),
			layoutName: "_rss",
			isImplicit: true,
		})
	}

	return err
}

func (cs *contents) setupImplicitPages() {
	root := filepath.Join(cs.s.cfg.Root, cs.s.cfg.PublicDir)

	for dir, hasIndex := range cs.dirHasIndex {
		if *hasIndex {
			continue
		}

		dst := filepath.Join(dir, "index.html")

		if _, ok := cs.dsts[dst]; ok {
			continue
		}

		base := fDropRoot(root, dst)

		f := file{
			srcPath:    filepath.Join(cs.s.cfg.Root, cs.s.cfg.ContentDir, base),
			dstPath:    base,
			layoutName: "_list",
			isImplicit: true,
		}

		if dir == root {
			f.layoutName = "_index"
		}

		cs.add(f)
	}

	cs.dirHasIndex = nil
}

func (cs *contents) find(c *content, currFile, rel string) (*content, error) {
	// No lock needed: find should only be called after ALL content has been
	// added

	src := ""
	if strings.HasPrefix(rel, p2ContentRelPfx) {
		rel = rel[len(p2ContentRelPfx):]
		src = filepath.Join(c.f.srcPath, "../", rel)
	} else {
		src = filepath.Join(currFile, "../", rel)
	}

	fc := cs.srcs[src]

	if fc == nil {
		return nil, fmt.Errorf("content `%s` from rel path `%s` not found", src, rel)
	}

	return fc, nil
}

func (cs *contents) claimDest(dst string, c *content) (
	alreadyClaimed bool,
	err error) {

	cs.mtx.Lock()

	if co, ok := cs.dsts[dst]; ok {
		if co == c {
			alreadyClaimed = true
		} else {
			err = fmt.Errorf("content conflict: destination file `%s` "+
				"already generated by `%s`",
				dst,
				co.f.srcPath)
		}
	} else {
		cs.dsts[dst] = c
	}

	cs.mtx.Unlock()

	return
}

func (c *content) load() error {
	mp := fChangeExt(c.f.srcPath, ".meta")
	if fExists(mp) {
		meta, err := ioutil.ReadFile(mp)
		if err != nil {
			return err
		}

		err = c.processMeta(meta, true)
		if err != nil {
			return err
		}
	}

	if c.f.isImplicit {
		return nil
	}

	f, err := os.Open(c.f.srcPath)
	if err != nil {
		return err
	}

	defer f.Close()
	r := bufio.NewReader(f)

	delim := make([]byte, len(metaDelim))
	i, err := r.Read(delim)
	if err != nil {
		if err != io.EOF {
			return err
		}

		return nil
	}

	if !bytes.HasPrefix(metaDelim, delim[:i]) {
		return nil
	}

	mb := &bytes.Buffer{}
	mb.Write(metaDelim)

	haveClosingDelim := false
	for err == nil {
		var l []byte
		l, err = r.ReadBytes('\n')
		mb.Write(l)

		if bytes.Equal(metaDelim, bytes.TrimSpace(l)) {
			haveClosingDelim = true
			break
		}
	}

	if !haveClosingDelim && err != nil {
		if err == io.EOF {
			err = fmt.Errorf("metadata missing closing `%s`", metaDelim)
		}

		return fmt.Errorf("failed to read content metadata: %v", err)
	}

	c.metaEnd = mb.Len()
	err = c.processMeta(mb.Bytes(), false)
	if err != nil {
		return err
	}

	lname := c.meta.layoutName()
	if lname != "" {
		lo := c.cs.s.findLayout(c.cpath, lname, false)
		if lo == nil {
			return fmt.Errorf("layout `%s` not found", lname)
		}

		c.f.layoutName = lname
	}

	return nil
}

func (c *content) processMeta(m []byte, isMetaFile bool) error {
	start := 3
	mt := metaYaml

	if !bytes.HasPrefix(m, metaDelim) {
		if !isMetaFile {
			return nil
		}

		start = 0
	} else {
		checkType := len(m) > metaHeaderTypeLen &&
			!bytes.Contains(m[:metaHeaderTypeLen], []byte("\n"))
		if checkType {
			dec := string(m[len(metaDelim)+1 : metaHeaderTypeLen])
			mt = metaTypeFromString(dec)
			if mt == metaUnknown {
				return fmt.Errorf("unrecognized metadata decoder: %s", dec)
			}

			start = metaHeaderTypeLen
		}
	}

	end := bytes.Index(m[3:], metaDelim)
	if end == -1 {
		if !isMetaFile {
			return nil
		}

		end = len(m)
	} else {
		end += 3
	}

	m = bytes.TrimSpace(m[start:end])
	err := c.meta.merge(m, mt)
	if err != nil {
		return fmt.Errorf("invalid metadata: %v", err)
	}

	return nil
}

func (c *content) shouldPublish() bool {
	publish, ok := c.meta.publish()
	if ok {
		return publish
	}

	isFuture := !c.tplCont.Date.IsZero() &&
		c.tplCont.Date.After(c.cs.s.stats.BuildStart)
	if isFuture && !c.cs.s.cfg.PublishFuture {
		return false
	}

	return true
}

func (c *content) analyze() {
	if c.deets.analyzed {
		return
	}

	ca := contentAnalyze{
		cfg:   c.cs.s.cfg,
		gen:   c.gen,
		deets: &c.deets,
	}
	ca.analyze()
}

func (c *content) getSummary() string {
	if len(c.deets.summary) > 0 {
		return c.deets.summary
	}

	sum := c.meta.summary()
	if len(sum) > 0 {
		c.deets.summary = sum
		return sum
	}

	c.analyze()
	return c.deets.summary
}

func (c *content) isChildOf(o *content) bool {
	return strings.HasPrefix(c.cpath, o.cpath+"/")
}

func (c *content) relDest(otherPath string) string {
	od := filepath.Dir(c.f.dstPath)
	d, f := filepath.Split(otherPath)

	rel, err := filepath.Rel(od, d)
	if err != nil {
		panic(err)
	}

	return filepath.Join(rel, f)
}

func (c *content) claimDest() (string, bool, error) {
	alreadyClaimed, err := c.cs.claimDest(c.f.dstPath, c)
	return c.f.dstPath, alreadyClaimed, err
}

func (c *content) claimOtherExt(ext string) (string, bool, error) {
	dst := fChangeExt(c.f.dstPath, ext)
	alreadyClaimed, err := c.cs.claimDest(dst, c)
	return dst, alreadyClaimed, err
}

func (c *content) templatize(w io.Writer) error {
	if c.f.isImplicit {
		return nil
	}

	b, err := ioutil.ReadFile(c.f.srcPath)
	if err != nil {
		return err
	}

	tpl, err := p2ContentSet.FromString(string(b[c.metaEnd:]))
	if err != nil {
		return err
	}

	return tpl.ExecuteWriter(c.tplCont.forPage(), w)
}

func (c *content) readAll(w io.Writer) error {
	if c.f.isImplicit {
		return nil
	}

	f, err := os.Open(c.f.srcPath)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Seek(int64(c.metaEnd), 0)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, f)
	return err
}
