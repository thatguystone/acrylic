package acrylib

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	p2 "github.com/flosch/pongo2"
)

type contents struct {
	s    *site
	mtx  sync.Mutex
	srcs map[string]*content // All available content: srcPath -> content
	dsts map[string]*content // All rendered content: dstPath -> content
}

type content struct {
	cs       *contents
	f        file
	cpath    string
	metaEnd  int
	meta     *meta
	lcctx    layoutContentCtx
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
	imgPubDir    = "img"
	layoutPubDir = "layout"
	themePubDir  = "theme"
)

var (
	metaDelim = []byte("---")

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
}

func (cs *contents) add(f file) error {
	if f.layoutName == "" {
		f.layoutName = "_single"
	}

	s := cs.s
	ext := filepath.Ext(f.srcPath)
	cpath := fChangeExt(f.dstPath, "")
	f.dstPath = filepath.Join(s.cfg.Root, s.cfg.PublicDir, f.dstPath)

	c := &content{
		cs:    cs,
		f:     f,
		cpath: cpath,
		meta:  &meta{},
	}

	c.gen = getContentGener(s, c, ext)

	err := c.load()
	if err != nil {
		return err
	}

	c.lcctx.init(cs.s, c)
	if !c.shouldPublish() {
		return nil
	}

	cs.mtx.Lock()
	cs.srcs[f.srcPath] = c
	cs.mtx.Unlock()

	cs.s.lsctx.addContentCtx(&c.lcctx)

	return nil
}

func (cs *contents) find(currFile, rel string) (*content, error) {
	// No lock needed: find should only be called after ALL content has been
	// added
	src := filepath.Join(currFile, "../", rel)
	c := cs.srcs[src]

	if c == nil {
		return nil, fmt.Errorf("content `%s` from rel path `%s` not found", src, rel)
	}

	return c, nil
}

func (cs *contents) claimDest(dst string, c *content) (alreadyClaimed bool, err error) {
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

	del := make([]byte, len(metaDelim))

	i, err := r.Read(del)
	if err != nil {
		if err != io.EOF {
			return err
		}

		return nil
	}

	if !bytes.Equal(metaDelim, del[:i]) {
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
	if !bytes.HasPrefix(m, metaDelim) {
		if !isMetaFile {
			return nil
		}
		start = 0
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
	return c.meta.merge(m)
}

func (c *content) shouldPublish() bool {
	publish, ok := c.meta.publish()
	if ok {
		return publish
	}

	isFuture := !c.lcctx.Date.IsZero() && c.lcctx.Date.After(c.cs.s.stats.BuildStart)
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

func (c *content) relDest(otherPath string) string {
	od := filepath.Dir(c.f.dstPath)
	d, f := filepath.Split(otherPath)

	rel, err := filepath.Rel(od, d)
	if err != nil {
		panic(err)
	}

	return filepath.Join(rel, f)
}

func (c *content) claimDest(ext string) (string, bool, error) {
	dst := c.f.dstPath
	if ext != "" {
		dst = fChangeExt(dst, ext)
	}

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

	return tpl.ExecuteWriter(c.lcctx.forPage(), w)
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
