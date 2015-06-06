package acryliclib

import (
	"errors"
	"fmt"
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
	s       *site
	c       *content
	Title   string
	Date    ctxTime
	summary string
	Meta    *meta
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
	return p2.Context{
		privSiteKey: lctx.s,
		contentKey:  lctx.c,
		assetOrdKey: &lctx.c.assetOrd,
		isPageKey:   true,
		// TODO(astone): need to restrict Site.Content.Find() in pages, too
		"Site": lctx.ls,
		"Page": layoutRestrictedPageCtx{lctx.lp},
	}
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

	ctx.summary = c.meta.summary()

	return &ctx
}

func (ctx *layoutPageCtx) Summary() *p2.Value {
	return p2.AsValue(ctx.c.getSummary())
}

func (ctx *layoutPageCtx) Content() *p2.Value {
	return p2.AsSafeValue(ctx.c.gen.getContent())
}

func (ctx *layoutRestrictedPageCtx) Summary() *p2.Value {
	ctx.s.errs.add(ctx.c.f.srcPath,
		// TODO(astone): add link to docs page explaining why
		errors.New("summaries of other pages may not be included in content"))
	return p2.AsValue("")
}

func (ctx *layoutRestrictedPageCtx) Content() *p2.Value {
	ctx.s.errs.add(ctx.c.f.srcPath,
		// TODO(astone): add link to docs page explaining why
		errors.New("content of other pages may not be included in content"))
	return p2.AsValue("")
}

func (t ctxTime) String() string {
	return t.Format(t.format)
}
