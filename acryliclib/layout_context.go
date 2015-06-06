package acryliclib

import (
	"errors"
	"path/filepath"
	"time"

	p2 "github.com/flosch/pongo2"
)

type layoutContext struct {
	s *site
	c *content

	ls *layoutSiteCtx
	lp *layoutPageCtx
}

type layoutSiteCtx struct {
	Name string
}

type layoutPageCtx struct {
	s     *site
	c     *content
	Title string
	Date  ctxTime
	Meta  *meta
}

type layoutRestrictedPageCtx struct {
	*layoutPageCtx
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

func newLayoutCtx(s *site, c *content) layoutContext {
	return layoutContext{
		s:  s,
		c:  c,
		ls: newLayoutSiteCtx(s),
		lp: newLayoutPageCtx(s, c),
	}
}

func (lctx layoutContext) forLayout(assetOrd *assetOrdering) p2.Context {
	return p2.Context{
		privSiteKey: lctx.s,
		contentKey:  lctx.c,
		assetOrdKey: assetOrd,
		"Site":      lctx.ls,
		"Page":      lctx.lp,
	}
}

func (lctx layoutContext) forPage() p2.Context {
	ctx := lctx.forLayout(&lctx.c.assetOrd)
	ctx[isPageKey] = true
	return ctx
}

func newLayoutSiteCtx(s *site) *layoutSiteCtx {
	ctx := layoutSiteCtx{}

	return &ctx
}

func newLayoutPageCtx(s *site, c *content) *layoutPageCtx {
	ctx := layoutPageCtx{
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

	return &ctx
}

func (ctx *layoutPageCtx) Summary(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	return p2.AsValue(ctx.c.getSummary())
}

func (ctx *layoutPageCtx) Content(tplP2Ctx *p2.ExecutionContext) *p2.Value {
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

func (t ctxTime) String() string {
	return t.Format(t.format)
}
