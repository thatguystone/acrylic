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

// TplSite contains the values exposed to templates as `Site`.
type TplSite struct {
	s     *site
	mtx   sync.Mutex
	Title string // Title of the site
	// Menu  TplMenu         // Menus available for use on the site
	Pages tplPageSlice // Sorted list of all pages
	Imgs  tplPageSlice // Sorted list of all images
	Blobs tplPageSlice // Sorted list of all blobs
}

// TplPage contains the values exposed to templates as `Page`.
type TplPage struct {
	s     *site
	c     *content
	Title string
	Date  tplTime
	Meta  *meta
}

// SEE FOR MENUS: http://gohugo.io/extras/menus/
// TplMenu contains menu information for a piece of content.
// type TplMenu struct {
// 	Name   string
// 	Weight int
// }

type tplPageSlice []*TplPage
type tplSiteMenu struct {
	tplPageSlice
}

type tplTime struct {
	time.Time
	format string
}

const (
	assetOrdKey = "__acrylicAssetsOrd__"
	contentKey  = "__acrylicContent__"
	isPageKey   = "__acrylicIsPage__"
	privSiteKey = "__acrylicSite__"
)

func newTplSite(s *site) *TplSite {
	tplSite := TplSite{
		s:     s,
		Title: s.cfg.Title,
	}

	return &tplSite
}

func (tplSite *TplSite) addContentCtx(tplCont *TplPage) {
	if tplCont.c.f.isImplicit {
		return
	}

	var lcs *tplPageSlice
	switch tplCont.c.gen.contType {
	case contPage:
		lcs = &tplSite.Pages

	case contImg:
		lcs = &tplSite.Imgs

	case contBlob:
		lcs = &tplSite.Blobs
	}

	if lcs == nil {
		return
	}

	tplSite.mtx.Lock()

	*lcs = append(*lcs, tplCont)

	tplSite.mtx.Unlock()
}

func (tplSite *TplSite) contentLoaded() {
	sort.Sort(tplSite.Pages)
	sort.Sort(tplSite.Imgs)
	sort.Sort(tplSite.Blobs)
	// sort.Sort(tplSiteMenu{tplSite.Menu})

	// fmt.Println("pages:", tplSite.Pages)
	// fmt.Println("imgs:", tplSite.Imgs)
	// fmt.Println("blobs:", tplSite.Blobs)
}

func (tplCont *TplPage) init(s *site, c *content) {
	tplCont.s = s
	tplCont.c = c
	tplCont.Meta = c.meta
	tplCont.Date = tplTime{format: s.cfg.DateFormat}

	if !c.f.isImplicit {
		base := filepath.Base(c.cpath)

		if date, ok := sToDate(base); ok {
			tplCont.Date.Time = date
			base = base[len(sDateFormat):]
		}

		tplCont.Title = sToTitle(base)
	}

	if title := c.meta.title(); title != "" {
		tplCont.Title = title
	}

	if date, ok := c.meta.date(); ok {
		tplCont.Date.Time = date
	}
}

func (tplCont *TplPage) forLayout(assetOrd *assetOrdering) p2.Context {
	return p2.Context{
		privSiteKey: tplCont.s,
		contentKey:  tplCont.c,
		assetOrdKey: assetOrd,
		"Site":      tplCont.s.tplSite,
		"Page":      tplCont,
	}
}

func (tplCont *TplPage) forPage() p2.Context {
	ctx := tplCont.forLayout(&tplCont.c.assetOrd)
	ctx[isPageKey] = true
	return ctx
}

// Summary gets a summary of the content
func (tplCont *TplPage) Summary(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	return p2.AsValue(tplCont.c.getSummary())
}

// Content gets an HTML dump of the content
func (tplCont *TplPage) Content(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	if _, ok := tplP2Ctx.Public[isPageKey]; ok {
		tplCont.s.errs.add(tplCont.c.f.srcPath,
			// TODO(astone): add link to docs page explaining why
			errors.New("content of other pages may not be included in other content"))
		return p2.AsValue("")
	}

	// Generate content first: this causes assets to be populated and
	// everything to be setup; once this returns, the content is ready for
	// use.
	html := tplCont.c.gen.getContent()

	ao := tplP2Ctx.Public[assetOrdKey].(*assetOrdering)
	ao.assimilate(&tplCont.s.assets, tplCont.c.assetOrd)

	return p2.AsSafeValue(html)
}

// IsActive determines if this content is the page currently being generated.
func (tplCont *TplPage) IsActive(tplP2Ctx *p2.ExecutionContext) bool {
	return false
}

// IsParentOf checks if the given content is a parent of this content.
func (tplCont *TplPage) IsParentOf(o *TplPage) bool {
	return false
}

func (ls tplPageSlice) Len() int      { return len(ls) }
func (ls tplPageSlice) Swap(a, b int) { ls[a], ls[b] = ls[b], ls[a] }

func (ls tplPageSlice) Less(a, b int) bool {
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

func (ls tplPageSlice) String() string {
	b := bytes.Buffer{}
	b.WriteRune('[')

	for i, tplCont := range ls {
		if i > 0 {
			b.WriteRune(' ')
		}

		path := tplCont.c.cpath
		date := tplCont.Date.Format(sDateFormat)
		if !tplCont.Date.IsZero() && !strings.Contains(path, date) {
			dir, file := filepath.Split(path)
			path = fmt.Sprintf("%s%s-%s", dir, date, file)
		}

		b.WriteString(path)
	}

	b.WriteRune(']')

	return b.String()
}

func (t tplTime) String() string {
	return t.Format(t.format)
}
