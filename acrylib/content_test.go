package acrylib

import (
	"fmt"
	"testing"
	"time"

	"github.com/thatguystone/assert"
)

func TestContentAutoMetas(t *testing.T) {
	t.Parallel()

	cfg := testConfig(false)
	cfg.DateFormat = "01-02-2006"

	pages := append([]testFile{}, basicSite...)
	tt := testNew(t, true, cfg, append(pages,
		testFile{
			p:  "layouts/date/_single.html",
			sc: "{{ Page.Date }} -- {{ Page.Title }}",
		},
		testFile{
			p:  "layouts/summary/_single.html",
			sc: `{{ Page.Summary }} -> content: {{ Page.Content }}`,
		},
		testFile{
			p:  "content/date/title-with_some-stuff-or-another.md",
			sc: "---\ndate: 2015-06-05\n---",
		},
		testFile{
			p: "content/date/2015-06-06-title-with-date.md",
		},
		testFile{
			p:  "content/date/2015-06-07.md",
			sc: "---\ntitle: stuffs and stuffs\n---",
		},
		testFile{
			p:  "content/summary/meta.md",
			sc: "---\nsummary: much summary\n---",
		},
		testFile{
			p:  "content/summary/content.md",
			sc: "much content",
		},
		testFile{
			p:  "content/summary/summary-more.md",
			sc: "i like my content\n\n<!--more-->\n\nbut you have to click to read more",
		},
	)...)
	defer tt.cleanup()

	tt.contents("date/title-with_some-stuff-or-another.html",
		`06-05-2015 -- Title with Some Stuff or Another`)

	tt.contents("date/2015-06-06-title-with-date.html",
		`06-06-2015 -- Title with Date`)

	tt.contents("date/2015-06-07.html",
		`06-07-2015 -- stuffs and stuffs`)

	tt.contents("summary/meta.html",
		`much summary -> content:`)

	tt.contents("summary/summary-more.html",
		`i like my content -> content:<p>i like my content</p><p>but you have to click to read more`)
}

func TestContentFuturePublishing(t *testing.T) {
	t.Parallel()

	future := time.Now().Add(time.Hour * 48).Format(sDateFormat)

	tt := testNew(t, true, nil,
		testFile{p: fmt.Sprintf("content/%s.md", future)},
		testFile{
			p:  "content/unpublished.md",
			sc: "---\npublish: false\n---",
		},
	)
	defer tt.cleanup()

	tt.notExists(fmt.Sprintf("%s.html", future))
	tt.notExists("unpublished.html")
}

func TestContentForcedPublish(t *testing.T) {
	t.Parallel()

	cfg := testConfig(false)
	cfg.PublishFuture = true

	future := time.Now().Add(time.Hour * 24).Format(sDateFormat)

	tt := testNew(t, true, cfg,
		testFile{p: fmt.Sprintf("content/%s.md", future)},
	)
	defer tt.cleanup()

	tt.exists(fmt.Sprintf("%s.html", future))
}

func TestContentIsChildOf(t *testing.T) {
	t.Parallel()
	a := assert.From(t)

	type test struct {
		child  string
		parent string
		is     bool
	}

	ca := &content{}
	cb := &content{}

	tests := []test{
		test{
			child:  "test/content/2",
			parent: "test/content",
			is:     true,
		},
		test{
			child:  "test/content",
			parent: "test/content/2",
			is:     false,
		},
		test{
			child:  "test/2015-06-05-test",
			parent: "test/2",
			is:     false,
		},
		test{
			child:  "test/2/015-06-05-test",
			parent: "test/2",
			is:     true,
		},
	}

	for _, t := range tests {
		ca.cpath = t.child
		cb.cpath = t.parent

		if t.is {
			a.True(ca.isChildOf(cb), "expected %s to be a child of %s",
				t.child,
				t.parent)
		} else {
			a.False(ca.isChildOf(cb), "did not expect %s to be a child of %s",
				t.child,
				t.parent)
		}
	}
}
