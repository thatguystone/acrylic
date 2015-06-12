package acrylib

import (
	"strings"
	"testing"
)

func TestSiteAssetCombining(t *testing.T) {
	t.Parallel()

	cfg := testConfig(false)
	cfg.RenderJS = true
	cfg.SingleJS = true
	cfg.RenderCSS = true
	cfg.SingleCSS = true

	pages := append([]testFile{}, basicSite...)
	tt := testNew(t, true, cfg, append(pages,
		testFile{
			p:  "lone/script.js",
			sc: `some page`,
		},
	)...)
	defer tt.cleanup()

	tt.exists("all.js")
	tt.notExists("layout/blog/layout2.js")

	tt.exists("all.css")
	tt.notExists("layout/blog/layout2.css")

	// The whole directory should be trashed
	tt.notExists("lone")
	tt.notExists("lone/script.js")

	tt.contents("blog/post2.html",
		"Blog layout:<h1>post 2</h1><p>post 2</p><img src=../../layout/blog/img.png style=width:1px;height:1px;><script src=../../../all.js></script><link rel=stylesheet href=../../../all.css>")

	fc := tt.readFile("all.js")
	tt.a.Log(fc)

	tt.a.Equal(1, strings.Count(fc, "(layout js)"), "js should only appear once")
	tt.a.Equal(1, strings.Count(fc, "(layout 2 js!)"), "js should only appear once")
	tt.a.Equal(1, strings.Count(fc, "(post 1 js)"), "js should only appear once")
	tt.a.Equal(1, strings.Count(fc, "(post 2 js)"), "js should only appear once")

	lojs := strings.Index(fc, "(layout js)")
	pjs := strings.Index(fc, "(post 1 js)")
	lo2js := strings.Index(fc, "(layout 2 js!)")
	tt.a.True(lojs < pjs, "layout js should be before post js: %d < %d", lojs, pjs)
	tt.a.True(pjs < lo2js, "post js should be before layout js2: %d < %d", pjs, lo2js)
}

func TestSiteAssetsOutOfOrder(t *testing.T) {
	t.Parallel()

	cfg := testConfig(false)
	cfg.RenderJS = true
	cfg.SingleJS = true
	cfg.RenderCSS = true
	cfg.SingleCSS = true

	tt := testNew(t, false, cfg,
		testFile{
			p: "content/blog/post1.md",
			sc: "# post 1\n" +
				"{% js \"post1.js\" %}\n" +
				"{% js \"post2.js\" %}\n",
		},
		testFile{
			p: "content/blog/post2.md",
			sc: "# post 2\n" +
				"{% js \"post2.js\" %}\n" +
				"{% js \"post1.js\" %}\n",
		},
		testFile{
			p:  "content/blog/post1.js",
			sc: "(post 1 js)",
		},
		testFile{
			p:  "content/blog/post2.js",
			sc: "(post 2 js)",
		})
	defer tt.cleanup()

	_, errs := Build(tt.cfg)
	tt.a.NotEqual(0, len(errs))

	es := errs.String()
	tt.a.True(strings.Contains(es, "asset ordering inconsistent"),
		"wrong error string: %s", es)
}

func TestSiteAssetsTrailer(t *testing.T) {
	t.Parallel()

	tt := testNew(t, true, nil,
		testFile{
			p: "content/all_assets.md",
			sc: "{% js \"coffee.coffee\" %}\n" +
				"{% css \"less.less\" %}\n",
		},
		testFile{p: "content/coffee.coffee"},
		testFile{p: "content/less.less"},
	)
	defer tt.cleanup()

	fc := tt.readFile("/all_assets.html")
	t.Log(fc)
	tt.a.True(strings.Contains(fc, tt.lastSite.cfg.LessURL))
	tt.a.True(strings.Contains(fc, tt.lastSite.cfg.CoffeeURL))
}

func TestSiteAssetMinify(t *testing.T) {
	t.Parallel()
	// TODO(astone): asset minification
}
