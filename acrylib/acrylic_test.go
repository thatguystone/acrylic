package acrylib

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thatguystone/assert"
)

type testAcrylic struct {
	a        assert.A
	cfg      Config
	lastSite *site
	isBench  bool
}

type testFile struct {
	dir bool
	p   string
	sc  string
	bc  []byte
}

var (
	gifBin = []byte{
		0x47, 0x49, 0x46, 0x38, 0x37, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x2c, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44, 0x01,
		0x00, 0x3b,
	}

	pngBin = []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00,
		0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde,
		0x00, 0x00, 0x00, 0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x0b,
		0x13, 0x00, 0x00, 0x0b, 0x13, 0x01, 0x00, 0x9a, 0x9c, 0x18, 0x00,
		0x00, 0x00, 0x07, 0x74, 0x49, 0x4d, 0x45, 0x07, 0xdf, 0x06, 0x03,
		0x16, 0x11, 0x34, 0xd8, 0x8f, 0x56, 0x73, 0x00, 0x00, 0x00, 0x19,
		0x74, 0x45, 0x58, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74,
		0x00, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x20, 0x77, 0x69,
		0x74, 0x68, 0x20, 0x47, 0x49, 0x4d, 0x50, 0x57, 0x81, 0x0e, 0x17,
		0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63,
		0xf8, 0xff, 0xff, 0x3f, 0x00, 0x05, 0xfe, 0x02, 0xfe, 0xdc, 0xcc,
		0x59, 0xe7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}

	basicSite = []testFile{
		testFile{
			p: "content/blog/post1.md",
			sc: "---\ntitle: test\n---\n" +
				"# post 1\n" +
				"{{ \"post 1\" }}\n" +
				"{% js \"post1.js\" %}\n" +
				"{% css \"post1.css\" %}\n",
		},
		testFile{
			p:  "content/blog/post1.js",
			sc: "(post 1 js)",
		},
		testFile{
			p:  "content/blog/post1.css",
			sc: "(post 1 css)",
		},
		testFile{
			p:  "content/blog/post1/stuff.html",
			sc: "(post 1 css)",
		},
		testFile{
			p: "content/blog/post2.md",
			sc: "---\ntitle: post2\n---\n" +
				"# post 2\n" +
				"{{ \"post 2\" }}\n" +
				"{% js \"post2.js\" %}\n" +
				"{% css \"post2.css\" %}\n",
		},
		testFile{
			p:  "content/blog/post2.js",
			sc: "(post 2 js)",
		},
		testFile{
			p:  "content/blog/post2.css",
			sc: "(post 2 css)",
		},
		testFile{
			p: "layouts/blog/_single.html",
			sc: "{% js \"layout.js\" %}\n" +
				"{% css \"layout.css\" %}\n" +
				"<body>" +
				"Blog layout: {{ Page.Content }}" +
				`{% img "img.png" %}` +
				"</body>" +
				"{% js_all %}\n" +
				"{% css_all %}\n" +
				"{% js \"layout2.js\" %}\n" +
				"{% css \"layout2.css\" %}\n",
		},
		testFile{
			p:  "layouts/blog/layout.js",
			sc: "(layout js)",
		},
		testFile{
			p:  "layouts/blog/layout.css",
			sc: "(layout css)",
		},
		testFile{
			p:  "layouts/blog/layout2.js",
			sc: "---\nrender: true\nval: layout 2\n---\n({{ Page.Meta.val }} js!)",
		},
		testFile{
			p:  "layouts/blog/layout2.css",
			sc: "({{ Page.Meta.val }} css!)",
		},
		testFile{
			p:  "layouts/blog/layout2.meta",
			sc: "render: true\nval: layout 2",
		},
		testFile{
			p:  "layouts/blog/img.png",
			bc: pngBin,
		},
		testFile{
			dir: true,
			p:   "content/blog/empty",
		},
	}
)

func testConfig(uglyURLs bool) *Config {
	cfg := &Config{
		Root:              filepath.Join("test_data", assert.GetTestName()),
		Title:             "test site",
		URL:               "http://example.com/site/",
		UglyURLs:          uglyURLs,
		MinifyHTML:        true,
		reproducibleBuild: true,
	}

	cfg.setDefaults()

	return cfg
}

func testNew(t testing.TB, build bool, cfg *Config, files ...testFile) *testAcrylic {
	if cfg == nil {
		cfg = testConfig(false)
	}

	tt := &testAcrylic{
		a:   assert.From(t),
		cfg: *cfg,
	}

	os.RemoveAll(tt.cfg.Root)
	tt.createFiles(files)

	_, tt.isBench = t.(*testing.B)
	if !tt.isBench {
		tt.a.Logf("Initial files:\n%s", tt.tree(""))
	}

	if build {
		tt.build()
	}

	return tt
}

func (tt *testAcrylic) build() BuildStats {
	site, stats, errs := build(tt.cfg)
	tt.a.MustEqual(0, len(errs), "failed to build site; errs=%v", errs)

	if !tt.isBench {
		tt.a.Logf("Generated files:\n%s", tt.tree(tt.cfg.PublicDir))
		tt.a.Logf("Page structure:\n%s", site.tplSite.Pages)
	}

	tt.lastSite = site

	return stats
}

func (tt *testAcrylic) createFiles(files []testFile) {
	for _, file := range files {
		p := filepath.Join(tt.cfg.Root, file.p)

		if file.dir {
			err := os.MkdirAll(p, 0700)
			tt.a.MustNotError(err, "failed to create dir %s", p)
		} else {
			var err error
			if len(file.sc) > 0 {
				err = fWrite(p, []byte(file.sc), 0640)
			} else {
				err = fWrite(p, file.bc, 0640)
			}

			tt.a.MustNotError(err, "failed to write file %s", p)
		}
	}
}

func (tt *testAcrylic) finalPath(path string) string {
	root := filepath.Join(tt.cfg.Root, tt.cfg.PublicDir)

	base := filepath.Base(path)
	if !tt.cfg.UglyURLs && strings.HasSuffix(path, ".html") && base != "index.html" {
		file := fChangeExt(base, "")
		path = filepath.Join(root, filepath.Dir(path), file, "index.html")
	} else {
		path = filepath.Join(root, path)
	}

	return path
}

func (tt *testAcrylic) exists(path string) {
	path = tt.finalPath(path)
	_, err := os.Stat(path)
	tt.a.True(err == nil, "file %s does not exist", path)
}

func (tt *testAcrylic) notExists(path string) {
	path = tt.finalPath(path)
	_, err := os.Stat(path)
	tt.a.False(err == nil, "file %s exists, but it shouldn't", path)
}

func (tt *testAcrylic) contents(path, contents string) {
	fc := tt.readFile(path)
	tt.a.Equal(contents, fc, "content mismatch for %s", path)
}

func (tt *testAcrylic) readFile(path string) string {
	path = tt.finalPath(path)
	f, err := os.Open(path)
	tt.a.MustNotError(err, "failed to open %s", path)
	defer f.Close()

	fc, err := ioutil.ReadAll(f)
	tt.a.MustNotError(err, "failed to read %s", path)

	return string(fc)
}

func (tt *testAcrylic) tree(dir string) string {
	root := filepath.Join(tt.cfg.Root, dir)

	if !dExists(root) {
		return ""
	}

	b := bytes.Buffer{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		b.WriteString("\t" + fDropRoot(tt.cfg.Root, path) + "\n")
		return nil
	})

	tt.a.MustNotError(err, "failed to walk %s", tt.cfg.Root)

	return b.String()
}

func (tt *testAcrylic) checkBinFile(path string, contents []byte) {

}

func (tt *testAcrylic) cleanup() {
	os.RemoveAll(tt.cfg.Root)
}

func TestEmptySite(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, nil)
	defer tt.cleanup()
}

func TestBasicSite(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, nil, basicSite...)
	defer tt.cleanup()

	tt.exists("blog/post1.js")
	tt.exists("blog/post2.js")
	tt.exists("blog/post1.css")
	tt.exists("blog/post2.css")
	tt.notExists("blog/empty/index.html")

	tt.contents("index.html", defaultLayouts["_index"])

	tt.contents("blog/post1.html",
		`<script src=../../layout/blog/layout.js></script><link rel=stylesheet href=../../layout/blog/layout.css>Blog layout:<h1>post 1</h1><p>post 1<script src=../post1.js></script><link rel=stylesheet href=../post1.css></p><img src=../../layout/blog/img.png style=width:1px;height:1px;><script src=../../layout/blog/layout2.js></script><link rel=stylesheet href=../../layout/blog/layout2.css>`)
	tt.contents("blog/post2.html",
		`<script src=../../layout/blog/layout.js></script><link rel=stylesheet href=../../layout/blog/layout.css>Blog layout:<h1>post 2</h1><p>post 2<script src=../post2.js></script><link rel=stylesheet href=../post2.css></p><img src=../../layout/blog/img.png style=width:1px;height:1px;><script src=../../layout/blog/layout2.js></script><link rel=stylesheet href=../../layout/blog/layout2.css>`)

	tt.contents("layout/blog/layout2.js",
		`(layout 2 js!)`)
	tt.contents("layout/blog/layout2.css",
		`(layout 2 css!)`)
}

func TestBasicSiteUglyURLs(t *testing.T) {
	t.Parallel()
	// tt := testNew(t, true, nil, basicSite...)
	// defer tt.cleanup()

	// TODO(astone): uglyURLs version of TestBasicSite
}

func TestIndexPagesInDirs(t *testing.T) {
	t.Parallel()
	// TODO(astone): test case for right layout chosen for an index.md/meta in a dir
}

func TestIndexContentConflict(t *testing.T) {
	t.Parallel()
	// TODO(astone): case for blog/post1.md && blog/post/index.html
}

func TestBuildStats(t *testing.T) {
	t.Parallel()

	pages := append([]testFile{}, basicSite...)
	tt := testNew(t, false, nil, append(pages,
		testFile{
			p:  "content/img_page.html",
			sc: `{% img "path.gif" %}`,
		},
		testFile{
			p:  "content/path.gif",
			bc: gifBin,
		},
	)...)
	defer tt.cleanup()

	stats := tt.build()
	tt.a.True(stats.Duration > 0,
		"duration wasn't set properly: %d == 0",
		stats.Duration)
	tt.a.True(stats.Pages > 0,
		"pages wasn't set properly: %d == 0",
		stats.Pages)
	tt.a.True(stats.JS > 0,
		"js wasn't set properly: %d == 0",
		stats.JS)
	tt.a.True(stats.CSS > 0,
		"css wasn't set properly: %d == 0",
		stats.CSS)
	tt.a.True(stats.Imgs > 0,
		"imgs wasn't set properly: %d == 0",
		stats.Imgs)
}

func TestLayoutAndThemesContentPages(t *testing.T) {
	t.Parallel()

	cfg := testConfig(false)
	cfg.Theme = "test"

	pages := append([]testFile{}, basicSite...)
	tt := testNew(t, true, cfg, append(pages,
		testFile{
			p:  "content/page.md",
			sc: `some page`,
		},
		testFile{
			p:  "layouts/_index.html",
			sc: `{% url "layout_linked.md" %}`,
		},
		testFile{
			p:  "layouts/layout_linked.md",
			sc: `layouts linked`,
		},
		testFile{
			p:  "layouts/layout_unlinked.md",
			sc: `layouts unlinked`,
		},
		testFile{
			p:  "themes/test/_single.html",
			sc: `{% url "theme_linked.md" %}`,
		},
		testFile{
			p:  "themes/test/theme_linked.md",
			sc: `theme linked`,
		},
		testFile{
			p:  "themes/test/theme_unlinked.md",
			sc: `theme unlinked`,
		},
	)...)
	defer tt.cleanup()

	tt.exists("layout/layout_linked.html")
	tt.exists("theme/test/theme_linked.html")

	tt.notExists("public/layout/layout_unlinked.html")
	tt.notExists("public/theme/test/theme_unlinked.html")
}

func TestSiteLayoutChanging(t *testing.T) {
	t.Parallel()

	content := "CRAZY COOL LAYOUT"

	tt := testNew(t, true, nil,
		testFile{
			p: "content/post.md",
			sc: "---\nlayoutName: /some/path/test\n---\n" +
				"this content shouldn't be displayed",
		},
		testFile{
			p:  "layouts/some/path/test.html",
			sc: content,
		},
	)
	defer tt.cleanup()

	tt.contents("post.html", content)
}

func TestRSS(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, nil,
		testFile{
			p:  "content/blog/index.meta",
			sc: "rss: true",
		},
		testFile{
			p:  "content/blog/2015-06-01-post0.md",
			sc: "content0",
		},
		testFile{
			p:  "content/blog/2015-06-02-post1.md",
			sc: "content1",
		},
		testFile{
			p:  "content/blog/2015-06-03-post2.md",
			sc: "content2",
		},
	)
	defer tt.cleanup()

	tt.contents("blog/feed.rss",
		`<rss version=2.0><channel><title>test site :: Feed</title><description></description><link>http://example.com/site/</link><pubdate>Mon, 01 Jan 0001 00:00:00 +0000</pubdate><item><title>Post2</title><description>content2</description><link>http:/example.com/site/blog/2015-06-03-post2</link><pubdate>Wed, 03 Jun 2015 00:00:00 -0400</pubdate></item><item><title>Post1</title><description>content1</description><link>http:/example.com/site/blog/2015-06-02-post1</link><pubdate>Tue, 02 Jun 2015 00:00:00 -0400</pubdate></item><item><title>Post0</title><description>content0</description><link>http:/example.com/site/blog/2015-06-01-post0</link><pubdate>Mon, 01 Jun 2015 00:00:00 -0400</pubdate></item></channel></rss>`)
}

func TestHighlight(t *testing.T) {
	t.Parallel()
	tt := testNew(t, true, nil,
		testFile{
			p:  "content/highlight.md",
			sc: "``` go\ntype a struct{}\n```\n",
			// `{% highlight "go" %}type a struct{}{% endhighlight %}`,
		})

	tt.contents("highlight.html", ``)
}

func BenchmarkEmptySite(b *testing.B) {
	tt := testNew(b, false, nil)
	defer tt.cleanup()

	for i := 0; i < b.N; i++ {
		tt.build()
	}
}

func BenchmarkBasicSite(b *testing.B) {
	tt := testNew(b, false, nil, basicSite...)
	defer tt.cleanup()

	for i := 0; i < b.N; i++ {
		tt.build()
	}
}
