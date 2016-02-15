package main

import (
	"sort"
	"time"
)

type pages struct {
	byCat map[string][]*page
}

type page struct {
	src        string
	dst        string
	sortName   string
	Cat        string
	Title      string
	Date       time.Time
	Content    string
	Summary    string
	URL        string
	isListPage bool
	Meta       map[string]interface{}
}

type pageSlice []*page

func (pgs *pages) add(p *page) {
	cat := pgs.byCat[p.Cat]
	cat = append(cat, p)
	pgs.byCat[p.Cat] = cat
}

func (pgs *pages) sort() {
	for _, pgs := range pgs.byCat {
		sort.Sort(pageSlice(pgs))
	}
}

func (pgs *pages) posts(cat string) []*page {
	var posts []*page

	add := func(ps []*page) {
		for _, p := range ps {
			if !p.Date.IsZero() {
				posts = append(posts, p)
			}
		}
	}

	if len(cat) == 0 {
		for _, ps := range pgs.byCat {
			add(ps)
		}

		sort.Sort(pageSlice(posts))
	} else {
		add(pgs.byCat[cat])
	}

	return posts
}

func (ps pageSlice) Len() int           { return len(ps) }
func (ps pageSlice) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }
func (ps pageSlice) Less(i, j int) bool { return ps[i].sortName > ps[j].sortName }
