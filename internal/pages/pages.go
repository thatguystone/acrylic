package pages

import (
	"sort"
	"sync"

	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/state"
)

type Ps struct {
	st *state.S

	mtx   sync.Mutex
	byCat map[string]pageSlice
}

type pageSlice []*P

func New(st *state.S) *Ps {
	return &Ps{
		st:    st,
		byCat: map[string]pageSlice{},
	}
}

func (ps *Ps) Load(f file.F) error {
	p, err := newP(ps.st, f)
	if err == nil {
		ps.mtx.Lock()

		cat := ps.byCat[p.Cat]
		cat = append(cat, p)
		ps.byCat[p.Cat] = cat

		ps.mtx.Unlock()
	}

	return err
}

func (ps *Ps) AllLoaded() {
	for _, pages := range ps.byCat {
		func(pages pageSlice) {
			ps.st.Run.Do(func() {
				sort.Sort(pages)
			})
		}(pages)
	}
}

func (ps *Ps) RenderPages(t TmplCompiler) {
	for _, cps := range ps.byCat {
		for _, p := range cps {
			if p.isListPage {
				continue
			}

			func(p *P) {
				ps.st.Run.Do(func() {
					err := p.Render(t)
					if err != nil {
						ps.st.Errs.Errorf(p.Src,
							"failed to render: %v",
							err)
					}
				})
			}(p)
		}
	}
}

func (ps *Ps) RenderListPages(t TmplCompiler) {
	for cat, cps := range ps.byCat {
		for _, p := range cps {
			if !p.isListPage {
				continue
			}

			func(cps []*P) {
				ps.st.Run.Do(func() {
					err := p.RenderList(t, cps)
					if err != nil {
						ps.st.Errs.Errorf(p.Src,
							"failed to category %s: %v",
							cat, err)
					}
				})
			}(cps)
		}
	}
}

func (ps *Ps) PostsIn(cat string) []*P {
	var posts pageSlice

	add := func(ps []*P) {
		for _, p := range ps {
			if !p.Date.IsZero() {
				posts = append(posts, p)
			}
		}
	}

	if len(cat) == 0 {
		for _, ps := range ps.byCat {
			add(ps)
		}

		sort.Sort(posts)
	} else {
		add(ps.byCat[cat])
	}

	return posts
}

func (ps pageSlice) Len() int           { return len(ps) }
func (ps pageSlice) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }
func (ps pageSlice) Less(i, j int) bool { return ps[i].SortName > ps[j].SortName }
