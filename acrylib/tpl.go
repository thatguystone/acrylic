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
	loaded bool
	s      *site
	mtx    sync.Mutex
	Title  string          // Title of the site
	Menus  TplMenus        // Menus available for use on the site
	Pages  tplContentSlice // Sorted list of all pages
	Imgs   tplContentSlice // Sorted list of all images
	Blobs  tplContentSlice // Sorted list of all blobs
}

type TplMenus map[string]tplContentSlice

// TplContent contains the values exposed to templates as `Page`.
type TplContent struct {
	s      *site
	c      *content
	Title  string          // Title of the page
	Date   tplTime         // Date included with content
	Meta   *meta           // Any fields put into any content metadata
	Childs tplContentSlice // Sub-content
	Weight int             // Weight for ordering
}

type tplContentSlice []*TplContent

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
		Menus: TplMenus{},
	}

	return &tplSite
}

func (tplSite *TplSite) addContent(tplCont *TplContent) error {
	// Pages added after site load are just auto-gen'd and aren't accessible
	// as content.
	if tplSite.loaded {
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
	*lcs = lcs.add(tplCont)
	tplSite.mtx.Unlock()

	err := tplSite.Menus.add(tplCont, &tplSite.mtx)
	if err != nil {
		err = fmt.Errorf("menu: %v", err)
	}

	return err
}

func (tplSite *TplSite) contentLoaded() {
	tplSite.loaded = true

	tplSite.Pages.sort()
	tplSite.Imgs.sort()
	tplSite.Blobs.sort()

	if len(tplSite.Menus) == 0 {
		// Use first-level content as main menu
	}

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
// content currently being generated.
func (tplCont *TplContent) IsChildActive(tplP2Ctx *p2.ExecutionContext) bool {
	c := tplP2Ctx.Public[contentKey].(*content)
	return tplCont.Active(tplP2Ctx) || c.isChildOf(tplCont.c)
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

func (s tplContentSlice) sort() {
	sort.Sort(s)
}

func (s tplContentSlice) Len() int      { return len(s) }
func (s tplContentSlice) Swap(a, b int) { s[a], s[b] = s[b], s[a] }

func (s tplContentSlice) Less(a, b int) bool {
	pa := s[a]
	pb := s[b]

	if pa.Weight != pb.Weight {
		return pa.Weight > pb.Weight
	}

	return pa.Less(pb)
}

func (s tplContentSlice) add(tplCont *TplContent) tplContentSlice {
	at := sort.Search(len(s), func(i int) bool {
		return s[i].c.cpath >= tplCont.c.cpath
	})

	checkInsert := func(i int) bool {
		if i < 0 || i >= len(s) {
			return false
		}

		occupied := s[i]

		switch {
		case occupied.c.isChildOf(tplCont.c):
			tplCont.Childs = tplCont.Childs.add(occupied)
			s[i] = tplCont

			// If any paths were added before a parent, push them down
			i++
			for i < len(s) {
				child := s[i]
				if !child.c.isChildOf(tplCont.c) {
					break
				}

				s = append(s[:i], s[i+1:]...)
				tplCont.Childs = tplCont.Childs.add(child)
			}

		case tplCont.c.isChildOf(occupied.c):
			occupied.Childs = occupied.Childs.add(tplCont)

		default:
			return false
		}

		return true
	}

	insert := !checkInsert(at) && !checkInsert(at-1)
	if insert {
		s = append(s, nil)
		copy(s[at+1:], s[at:])
		s[at] = tplCont
	}

	return s
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

		if len(tplCont.Childs) > 0 {
			b.WriteRune('>')
			b.WriteString(tplCont.Childs.String())
		}
	}

	b.WriteRune(']')

	return b.String()
}

func (ms TplMenus) add(tplCont *TplContent, mtx *sync.Mutex) error {
	mm := tplCont.Meta.menu()
	if mm == nil {
		return nil
	}

	mCont := *tplCont
	mCont.Childs = nil

	add := func(k string) {
		mtx.Lock()
		ms[k] = ms[k].add(&mCont)
		mtx.Unlock()
	}

	rv := reflect.ValueOf(mm)
	switch rv.Kind() {
	case reflect.String:
		add(mm.(string))

	case reflect.Slice:
		for _, vi := range mm.([]interface{}) {
			if v, ok := vi.(string); !ok {
				return fmt.Errorf("values in array must be strings, not %v=%v",
					reflect.TypeOf(vi), vi)
			} else {
				add(v)
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
				add(k)
				continue
			}

			kv = reflect.ValueOf(kv.Interface())
			if kv.Kind() != reflect.Map {
				return fmt.Errorf("menu values must be maps, not %v=%v",
					kv.Type(), kv.Interface())
			}

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
					mCont.Title, ok = optVi.(string)
					if !ok {
						return fmt.Errorf("title key must have a string value, "+
							"not %v=%v",
							optV.Type(), optVi)
					}

				case "weight":
					switch wv := optVi.(type) {
					case int:
						mCont.Weight = wv

					case float64:
						mCont.Weight = int(wv)

					default:
						return fmt.Errorf("weight key must have an integer value, "+
							"not %v=%v",
							optV.Type(), optVi)
					}
				}
			}

			add(k)
		}
	}

	return nil
}

func (t tplTime) String() string {
	return t.Format(t.format)
}
