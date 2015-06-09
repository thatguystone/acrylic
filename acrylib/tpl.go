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
	Title string              // Title of the site
	Menus map[string]*TplMenu // Menus available for use on the site
	Pages tplContentSlice     // Sorted list of all pages
	Imgs  tplContentSlice     // Sorted list of all images
	Blobs tplContentSlice     // Sorted list of all blobs
}

// TplContent contains the values exposed to templates as `Page`.
type TplContent struct {
	s     *site
	c     *content
	Title string
	Date  tplTime
	Meta  *meta
}

// TplMenu contains site-wide menuing information.
type TplMenu struct {
	Name     string
	Links    tplMenuContentSlice
	SubMenus map[string]*TplMenu
}

// TplMenuContent contains menu information for a piece of content.
type TplMenuContent struct {
	Name   string // The name given in the meta or the page's Title
	Page   *TplContent
	Weight int
}

type tplContentSlice []*TplContent
type tplMenuContentSlice []*TplMenuContent

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
		Menus: map[string]*TplMenu{},
	}

	return &tplSite
}

func (tplSite *TplSite) addContent(tplCont *TplContent) {
	if tplCont.c.f.isImplicit {
		return
	}

	var lcs *tplContentSlice
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

	for _, m := range tplSite.Menus {
		m.sort()
	}

	// fmt.Println("pages:", tplSite.Pages)
	// fmt.Println("imgs:", tplSite.Imgs)
	// fmt.Println("blobs:", tplSite.Blobs)
}

func (tplCont *TplContent) init(s *site, c *content) {
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

func (tplCont *TplContent) forLayout(assetOrd *assetOrdering) p2.Context {
	return p2.Context{
		privSiteKey: tplCont.s,
		contentKey:  tplCont.c,
		assetOrdKey: assetOrd,
		"Site":      tplCont.s.tplSite,
		"Page":      tplCont,
	}
}

func (tplCont *TplContent) forPage() p2.Context {
	ctx := tplCont.forLayout(&tplCont.c.assetOrd)
	ctx[isPageKey] = true
	return ctx
}

// Summary gets a summary of the content
func (tplCont *TplContent) Summary(tplP2Ctx *p2.ExecutionContext) *p2.Value {
	return p2.AsValue(tplCont.c.getSummary())
}

// Content gets an HTML dump of the content
func (tplCont *TplContent) Content(tplP2Ctx *p2.ExecutionContext) *p2.Value {
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
func (tplCont *TplContent) IsActive(tplP2Ctx *p2.ExecutionContext) bool {
	return false
}

// IsParentOf checks if the given content is a parent of this content.
func (tplCont *TplContent) IsParentOf(o *TplContent) bool {
	return false
}

func (tplCont *TplContent) Less(o *TplContent) bool {
	ap, af := filepath.Split(tplCont.c.cpath)
	bp, bf := filepath.Split(o.c.cpath)

	if ap == bp {
		if tplCont.Date.Equal(o.Date.Time) {
			return af < bf

		}

		return tplCont.Date.After(o.Date.Time)
	}

	return ap < bp
}

func (s tplContentSlice) Len() int      { return len(s) }
func (s tplContentSlice) Swap(a, b int) { s[a], s[b] = s[b], s[a] }

func (s tplContentSlice) Less(a, b int) bool {
	return s[a].Less(s[b])
}

func (s tplContentSlice) String() string {
	b := bytes.Buffer{}
	b.WriteRune('[')

	for i, tplCont := range s {
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

func (m *TplMenu) sort() {
	sort.Sort(m.Links)

	for _, sm := range m.SubMenus {
		sm.sort()
	}
}

func (s tplMenuContentSlice) Len() int      { return len(s) }
func (s tplMenuContentSlice) Swap(a, b int) { s[a], s[b] = s[b], s[a] }

func (s tplMenuContentSlice) Less(a, b int) bool {
	pa := s[a]
	pb := s[b]

	if pa.Weight > 0 || pb.Weight > 0 || pa.Weight != pb.Weight {
		return pa.Weight < pb.Weight
	}

	return pa.Page.Less(pb.Page)
}

func (t tplTime) String() string {
	return t.Format(t.format)
}
