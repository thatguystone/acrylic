package acrylib

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
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
	Title string          // Title of the site
	Menus TplMenus        // Menus available for use on the site
	Pages tplContentSlice // Sorted list of all pages
	Imgs  tplContentSlice // Sorted list of all images
	Blobs tplContentSlice // Sorted list of all blobs
}

type TplMenus map[string][]*TplMenuContent

// TplContent contains the values exposed to templates as `Page`.
type TplContent struct {
	s     *site
	c     *content
	Title string  // Title of the page
	Date  tplTime // Date included with content
	Meta  *meta   // Any fields put into any content metadata
}

// TplMenuContent contains menu information for a piece of content.
type TplMenuContent struct {
	menuName string
	Page     *TplContent
	Childs   []*TplMenuContent
	TplMenuOpts
}

// TplMenuOpts contains the options that may be used when creating a menu in
// metadata.
type TplMenuOpts struct {
	Title  string // The name given in the meta or the page's Title
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
		Menus: map[string][]*TplMenuContent{},
	}

	return &tplSite
}

func (tplSite *TplSite) addContent(tplCont *TplContent) error {
	if tplCont.c.f.isImplicit {
		return nil
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
		return nil
	}

	tplSite.mtx.Lock()

	*lcs = append(*lcs, tplCont)

	tplSite.mtx.Unlock()

	err := tplSite.Menus.add(tplCont, &tplSite.mtx)
	if err != nil {
		err = fmt.Errorf("menu: %v", err)
	}

	return err
}

func (tplSite *TplSite) contentLoaded() {
	sort.Sort(tplSite.Pages)
	sort.Sort(tplSite.Imgs)
	sort.Sort(tplSite.Blobs)

	if len(tplSite.Menus) == 0 {
		// Use first-level content as main menu
	}

	for _, m := range tplSite.Menus {
		tplMenuContentSlice(m).sort()
	}

	// fmt.Println("pages:", tplSite.Pages)
	// fmt.Println("imgs:", tplSite.Imgs)
	// fmt.Println("blobs:", tplSite.Blobs)
}

func (tplCont *TplContent) init(s *site, c *content) {
	tplCont.s = s
	tplCont.c = c
	tplCont.Date = tplTime{format: s.cfg.DateFormat}

	tplCont.Meta = c.meta

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
func (tplCont *TplContent) Active(tplP2Ctx *p2.ExecutionContext) bool {
	return tplCont.c == tplP2Ctx.Public[contentKey]
}

// IsChildActive determines if this content or any of its sub-content is the
// page currently being generated.
func (tplCont *TplContent) IsChildActive(tplP2Ctx *p2.ExecutionContext) bool {
	c := tplP2Ctx.Public[contentKey].(*content)
	return strings.HasPrefix(c.cpath, tplCont.c.cpath)
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

func (tmc *TplMenuContent) Active(tplP2Ctx *p2.ExecutionContext) bool {
	return tmc.Page.Active(tplP2Ctx)
}

func (tmc *TplMenuContent) IsChildActive(tplP2Ctx *p2.ExecutionContext) bool {
	return tmc.Page.IsChildActive(tplP2Ctx)
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

func (ms TplMenus) add(tplCont *TplContent, mtx *sync.Mutex) error {
	mm := tplCont.Meta.menu()
	if mm == nil {
		return nil
	}

	var insertInto func(mCont *TplMenuContent, slice []*TplMenuContent) []*TplMenuContent
	insertInto = func(mCont *TplMenuContent, slice []*TplMenuContent) []*TplMenuContent {
		at := sort.Search(len(slice), func(i int) bool {
			return slice[i].Page.c.cpath >= mCont.Page.c.cpath
		})

		myCPath := mCont.Page.c.cpath
		checkInsert := func(i int) bool {
			if i < 0 || i >= len(slice) {
				return false
			}

			occupied := slice[i]
			occupiedCPath := occupied.Page.c.cpath

			switch {
			case strings.HasPrefix(occupiedCPath, myCPath):
				mCont.Childs = insertInto(occupied, mCont.Childs)
				slice[i] = mCont

				// If any paths were added before a parent, push them down
				i++
				for i < len(slice) {
					child := slice[i]
					if !strings.HasPrefix(child.Page.c.cpath, myCPath) {
						break
					}

					slice = append(slice[:i], slice[i+1:]...)
					mCont.Childs = insertInto(child, mCont.Childs)
				}

			case strings.HasPrefix(myCPath, occupiedCPath):
				occupied.Childs = insertInto(mCont, occupied.Childs)

			default:
				return false
			}

			return true
		}

		insert := !checkInsert(at) && !checkInsert(at-1)
		if insert {
			slice = append(slice, nil)
			copy(slice[at+1:], slice[at:])
			slice[at] = mCont
		}

		return slice
	}

	addOpts := func(k string, opts TplMenuOpts) {
		mCont := &TplMenuContent{
			menuName:    k,
			Page:        tplCont,
			TplMenuOpts: opts,
		}

		mtx.Lock()

		menus := ms[k]
		ms[k] = insertInto(mCont, menus)

		mtx.Unlock()
	}

	addString := func(k string) {
		addOpts(k, TplMenuOpts{
			Title:  tplCont.Title,
			Weight: 0,
		})
	}

	rv := reflect.ValueOf(mm)
	switch rv.Kind() {
	case reflect.String:
		addString(mm.(string))

	case reflect.Slice:
		for _, vi := range mm.([]interface{}) {
			if v, ok := vi.(string); !ok {
				return fmt.Errorf("values in array must be strings, not %v=%v",
					reflect.TypeOf(vi), vi)
			} else {
				addString(v)
			}
		}

	case reflect.Map:
		keys := rv.MapKeys()
		for _, kv := range keys {
			k, ok := kv.Interface().(string)
			if !ok {
				return fmt.Errorf("keys in map must be strings, not %v=%v",
					kv.Type(), kv.Interface())
			}

			kv = rv.MapIndex(kv)
			if kv.IsNil() {
				addString(k)
				continue
			}

			kv = reflect.ValueOf(kv.Interface())
			if kv.Kind() != reflect.Map {
				return fmt.Errorf("menu values must be maps, not %v=%v",
					kv.Type(), kv.Interface())
			}

			c := TplMenuOpts{}
			opts := kv.MapKeys()
			for _, optKv := range opts {
				optK, ok := optKv.Interface().(string)
				if !ok {
					return fmt.Errorf("menu keys must be strings, not %v=%v",
						kv.Type(), kv.Interface())
				}

				optV := kv.MapIndex(optKv)
				optVi := optV.Interface()
				switch strings.ToLower(optK) {
				case "title":
					c.Title, ok = optVi.(string)
					if !ok {
						return fmt.Errorf("title key must have a string value, not %v=%v",
							optV.Type(), optVi)
					}

				case "weight":
					switch wv := optVi.(type) {
					case int:
						c.Weight = wv

					case float64:
						c.Weight = int(wv)

					default:
						return fmt.Errorf("weight key must have an integer value, "+
							"not %v=%v",
							optV.Type(), optVi)
					}
				}
			}

			addOpts(k, c)
		}
	}

	return nil
}

func (s tplMenuContentSlice) sort() {
	sort.Sort(s)

	for _, sm := range s {
		tplMenuContentSlice(sm.Childs).sort()
	}
}

func (s tplMenuContentSlice) Len() int      { return len(s) }
func (s tplMenuContentSlice) Swap(a, b int) { s[a], s[b] = s[b], s[a] }

func (s tplMenuContentSlice) Less(a, b int) bool {
	pa := s[a]
	pb := s[b]

	if pa.Weight != pb.Weight {
		return pa.Weight > pb.Weight
	}

	return pa.Page.Less(pb.Page)
}

func (t tplTime) String() string {
	return t.Format(t.format)
}
