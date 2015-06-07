package acrylib

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	p2 "github.com/flosch/pongo2"
)

type layoutSiteCtx struct {
	s     *site
	mtx   sync.Mutex
	Title string
	Pages layoutContentCtxSlice
	Imgs  layoutContentCtxSlice
}

type layoutContentCtx struct {
	s     *site
	c     *content
	Title string
	Date  ctxTime
	Meta  *meta
}

type layoutContentCtxSlice []*layoutContentCtx

type ctxTime struct {
	time.Time
	format string
}

const (
	assetOrdKey = "__acrylicAssetsOrd__"
	contentKey  = "__acrylicContent__"
	isPageKey   = "__acrylicIsPage__"
	privSiteKey = "__acrylicSite__"
)

func newLayoutSiteCtx(s *site) *layoutSiteCtx {
	lsctx := layoutSiteCtx{
		s:     s,
		Title: s.cfg.Title,
	}

	return &lsctx
}

func (lsctx *layoutSiteCtx) addContentCtx(lcctx *layoutContentCtx) {
	if lcctx.c.f.isImplicit {
		return
	}

	var lcs *layoutContentCtxSlice
	switch lcctx.c.gen.contType {
	case contPage:
		lcs = &lsctx.Pages

	case contImg:
		lcs = &lsctx.Imgs
	}

	if lcs == nil {
		return
	}

	lsctx.mtx.Lock()

	*lcs = append(*lcs, lcctx)

	lsctx.mtx.Unlock()
}

func (lsctx *layoutSiteCtx) contentLoaded() {
	sort.Sort(lsctx.Pages)
	sort.Sort(lsctx.Imgs)

	// fmt.Println(lsctx.Pages)
	// fmt.Println(lsctx.Imgs)
}

func newLayoutContentCtx(s *site, c *content) *layoutContentCtx {
	lcctx := &layoutContentCtx{
		s:    s,
		c:    c,
		Meta: c.meta,
		Date: ctxTime{format: s.cfg.DateFormat},
	}

	if !c.f.isImplicit {
		base := filepath.Base(c.cpath)

		if date, ok := sToDate(base); ok {
			lcctx.Date.Time = date
			base = base[len(sDateFormat):]
		}

		lcctx.Title = sToTitle(base)
	}

	if title := c.meta.title(); title != "" {
		lcctx.Title = title
	}

	if date, ok := c.meta.date(); ok {
		lcctx.Date.Time = date
	}

	s.lsctx.addContentCtx(lcctx)

	return lcctx
}

func (lcctx *layoutContentCtx) forLayout(assetOrd *assetOrdering) p2.Context {
	return p2.Context{
		privSiteKey: lcctx.s,
		contentKey:  lcctx.c,
		assetOrdKey: assetOrd,
		"Site":      lcctx.s.lsctx,
		"Page":      lcctx,
	}
}

func (lcctx *layoutContentCtx) forPage() p2.Context {
	ctx := lcctx.forLayout(&lcctx.c.assetOrd)
	ctx[isPageKey] = true
	return ctx
}

func (lcctx *layoutContentCtx) Summary(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	return p2.AsValue(lcctx.c.getSummary())
}

func (lcctx *layoutContentCtx) Content(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	if _, ok := tplP2Ctx.Public[isPageKey]; ok {
		lcctx.s.errs.add(lcctx.c.f.srcPath,
			// TODO(astone): add link to docs page explaining why
			errors.New("content of other pages may not be included in other content"))
		return p2.AsValue("")
	}

	// Generate content first: this causes assets to be populated and
	// everything to be setup; once this returns, the content is ready for
	// use.
	html := lcctx.c.gen.getContent()

	ao := tplP2Ctx.Public[assetOrdKey].(*assetOrdering)
	ao.assimilate(&lcctx.s.assets, lcctx.c.assetOrd)

	return p2.AsSafeValue(html)
}

func (lcctx *layoutContentCtx) IsActive(tplP2Ctx *p2.ExecutionContext) bool {
	return false
}

func (lcctx *layoutContentCtx) IsParent(o *layoutContentCtx) bool {
	return false
}

func (ls layoutContentCtxSlice) Len() int      { return len(ls) }
func (ls layoutContentCtxSlice) Swap(a, b int) { ls[a], ls[b] = ls[b], ls[a] }

func (ls layoutContentCtxSlice) Less(a, b int) bool {
	actx := ls[a]
	bctx := ls[b]
	ap, af := filepath.Split(actx.c.cpath)
	bp, bf := filepath.Split(bctx.c.cpath)

	if ap == bp {
		if actx.Date.Equal(bctx.Date.Time) {
			return af < bf

		}

		return actx.Date.After(bctx.Date.Time)
	}

	return ap < bp
}

func (ls layoutContentCtxSlice) String() string {
	b := bytes.Buffer{}
	b.WriteRune('[')

	for i, lcctx := range ls {
		if i > 0 {
			b.WriteRune(' ')
		}

		path := lcctx.c.cpath
		date := lcctx.Date.Format(sDateFormat)
		if !lcctx.Date.IsZero() && !strings.Contains(path, date) {
			dir, file := filepath.Split(path)
			path = fmt.Sprintf("%s%s-%s", dir, date, file)
		}

		b.WriteString(path)
	}

	b.WriteRune(']')

	return b.String()
}

func (t ctxTime) String() string {
	return t.Format(t.format)
}
