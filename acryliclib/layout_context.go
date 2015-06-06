package acryliclib

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	p2 "github.com/flosch/pongo2"
)

type layoutSiteCtx struct {
	s       *site
	mtx     sync.Mutex
	Name    string
	Content []*layoutContentCtx
}

type layoutContentCtx struct {
	s     *site
	c     *content
	Title string
	Date  ctxTime
	Meta  *meta
}

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
	ctx := layoutSiteCtx{
		s: s,
	}

	return &ctx
}

func (lsctx *layoutSiteCtx) addContentCtx(lcctx *layoutContentCtx) {
	lsctx.mtx.Lock()

	lsctx.Content = append(lsctx.Content, lcctx)

	lsctx.mtx.Unlock()
}

func newLayoutContentCtx(s *site, c *content) *layoutContentCtx {
	ctx := &layoutContentCtx{
		s:    s,
		c:    c,
		Meta: c.meta,
		Date: ctxTime{format: s.cfg.DateFormat},
	}

	if !c.f.isImplicit {
		base := filepath.Base(c.cpath)

		if date, ok := sToDate(base); ok {
			ctx.Date.Time = date
			base = base[len(sDateFormat):]
		}

		ctx.Title = sToTitle(base)
	}

	if title := c.meta.title(); title != "" {
		ctx.Title = title
	}

	if date, ok := c.meta.date(); ok {
		ctx.Date.Time = date
	}

	s.lsctx.addContentCtx(ctx)

	return ctx
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

func (ctx *layoutContentCtx) Summary(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	return p2.AsValue(ctx.c.getSummary())
}

func (ctx *layoutContentCtx) Content(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	if _, ok := tplP2Ctx.Public[isPageKey]; ok {
		ctx.s.errs.add(ctx.c.f.srcPath,
			// TODO(astone): add link to docs page explaining why
			errors.New("content of other pages may not be included in other content"))
		return p2.AsValue("")
	}

	// Generate content first: this causes assets to be populated and
	// everything to be setup; once this returns, the content is ready for
	// use.
	html := ctx.c.gen.getContent()

	ao := tplP2Ctx.Public[assetOrdKey].(*assetOrdering)
	ao.assimilate(&ctx.s.assets, ctx.c.assetOrd)

	return p2.AsSafeValue(html)
}

func (ctx *layoutContentCtx) IsActive(tplP2Ctx *p2.ExecutionContext) bool {
	return false
}

func (t ctxTime) String() string {
	return t.Format(t.format)
}
