package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/thatguystone/assert"
)

type test struct {
	a    assert.A
	root string
}

type testFile struct {
	p  string
	sc string
	bc []byte
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
)

func testNew(tb testing.TB, cfgs []string, files ...testFile) *test {
	tt := test{
		a:    assert.From(tb),
		root: filepath.Join("test_data", assert.GetTestName()),
	}

	os.RemoveAll(tt.root)

	for _, f := range files {
		p := filepath.Join(tt.root, f.p)

		var err error
		if len(f.sc) > 0 {
			err = fWrite(p, []byte(f.sc))
		} else {
			err = fWrite(p, f.bc)
		}

		tt.a.MustNotError(err)
	}

	for i, cfg := range cfgs {
		cfgs[i] = filepath.Join(tt.root, cfg)
	}

	b := bytes.Buffer{}
	run(cfgs, tt.root, &b, true)
	if b.Len() > 0 {
		tb.Log(b.String())
		tb.FailNow()
	}

	tt.a.Logf("Generated files:\n%s", tt.tree(tt.root, "public/"))

	return &tt
}

func (t *test) cleanup() {
	os.RemoveAll(t.root)
}

func (t *test) finalPath(path string) string {
	return filepath.Join(t.root, path)
}

func (t *test) contents(path, c string) {
	fc, err := ioutil.ReadFile(t.finalPath(path))
	t.a.MustNotError(err)

	t.a.Equal(c, string(fc), "content mismatch for %s", path)
}

func (t *test) tree(root, dir string) string {
	root = filepath.Join(root, dir)

	if !dExists(root) {
		return ""
	}

	b := bytes.Buffer{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		b.WriteString("\t" + fDropRoot(root, "", path) + "\n")
		return nil
	})

	t.a.MustNotError(err, "failed to walk %s", root)

	return b.String()
}

func TestBasic(t *testing.T) {
	t.Parallel()
	tt := testNew(t, []string{"conf.yml"},
		testFile{
			p: "conf.yml",
			sc: "debug: true\n" +
				"js:\n" +
				"    - js/one.js\n" +
				"css:\n" +
				"    - css/one.scss\n",
		},
		testFile{
			p:  "content/index.html",
			sc: `{% extends "base.html" %}{% block content %}test{% endblock %}`,
		},
		testFile{
			p:  "content/robots.txt",
			sc: `robots!`,
		},
		testFile{
			p: "content/blog/index.html",
			sc: "---\nlist_page: true\n---\n" +
				`|{% for p in pages %} {{ p.Title }} |{% endfor %}`,
		},
		testFile{
			p:  "content/blog/2015-01-05-post0.html",
			sc: "{{ data.test.val|integer }}",
		},
		testFile{
			p: "content/blog/2015-01-07-post1/index.html",
			sc: `{{ ac.Img("test.png").Scale(5, 0, false, 90) }}` + "\n" +
				`{{ ac.Img("test.png").Scale(0, 0, false, 100) }}`,
		},
		testFile{
			p:  "content/blog/2015-01-07-post1/test.png",
			bc: pngBin,
		},
		testFile{
			p:  "data/test",
			sc: `{"val": 1234}`,
		},
		testFile{
			p:  "assets/js/one.js",
			sc: `function() {};`,
		},
		testFile{
			p:  "assets/css/one.scss",
			sc: `body { background: #000; }`,
		},
		testFile{
			p:  "templates/base.html",
			sc: `{% block header %}HEADER{% endblock %} {% block content %}WRONG{% endblock %}`,
		},
	)
	defer tt.cleanup()

	tt.contents("public/index.html", "HEADER test")
	tt.contents("public/robots.txt", "robots!")
	tt.contents("public/assets/js/one.js", "function() {};")
	tt.contents("public/assets/css/one.css", "body {\n  background: #000; }\n")
	tt.contents("public/blog/index.html", "| Post1 | Post0 |")
	tt.contents("public/blog/2015/01/05/post0/index.html", "1234")
	tt.contents("public/blog/2015/01/07/post1/index.html",
		"/blog/2015/01/07/post1/test.5x-q90.png\n/blog/2015/01/07/post1/test.png")
}

func TestPublish(t *testing.T) {
	t.Parallel()
	tt := testNew(t, []string{"conf.yml"},
		testFile{
			p: "conf.yml",
			sc: "debug: false\n" +
				"js:\n" +
				"    - js/one.js\n" +
				"css:\n" +
				"    - css/one.scss\n",
		},
		testFile{
			p:  "content/index.html",
			sc: "<html>\n\n\n\n\n\ntest\n\n\n\n\n</html>",
		},
		testFile{
			p:  "assets/js/one.js",
			sc: `function(test) { ; } ("abced");`,
		},
		testFile{
			p:  "assets/css/one.scss",
			sc: `body { background: #000; }`,
		},
	)
	defer tt.cleanup()

	tt.contents("public/index.html", "test")
	tt.contents("public/assets/all.js", `function(test){;}("abced");`)
	tt.contents("public/assets/all.css", "body{background:#000}")
}

func TestAssetTagsDebug(t *testing.T) {
	t.Parallel()

	tt := testNew(t, []string{"conf.yml"},
		testFile{
			p: "conf.yml",
			sc: "debug: true\n" +
				"js:\n" +
				"    - js/one.js\n" +
				"css:\n" +
				"    - css/one.scss\n",
		},
		testFile{
			p:  "content/index.html",
			sc: "{{ ac.JSTags() }}{{ ac.CSSTags() }}",
		},
		testFile{
			p: "assets/js/one.js",
		},
		testFile{
			p:  "assets/css/one.scss",
			sc: `body { background: #000; }`,
		},
	)
	defer tt.cleanup()

	tt.contents("public/index.html",
		`<script type="text/javascript" src="/assets/js/one.js"></script><link rel="stylesheet" href="/assets/css/one.css" />`)
}

func TestAssetTagsPublish(t *testing.T) {
	t.Parallel()

	tt := testNew(t, []string{"conf.yml"},
		testFile{
			p: "conf.yml",
			sc: "debug: false\n" +
				"js:\n" +
				"    - js/one.js\n" +
				"css:\n" +
				"    - css/one.scss\n",
		},
		testFile{
			p:  "content/index.html",
			sc: "{{ ac.JSTags() }}{{ ac.CSSTags() }}",
		},
		testFile{
			p: "assets/js/one.js",
		},
		testFile{
			p:  "assets/css/one.scss",
			sc: `body { background: #000; }`,
		},
	)
	defer tt.cleanup()

	tt.contents("public/index.html",
		`<script src=/assets/all.js></script><link rel=stylesheet href=/assets/all.css>`)
}

func TestCSSAssets(t *testing.T) {
	t.Parallel()
	tt := testNew(t, []string{"conf.yml"},
		testFile{
			p: "conf.yml",
			sc: "debug: true\n" +
				"css:\n" +
				"    - css/one.scss\n",
		},
		testFile{
			p:  "assets/img/test.png",
			bc: pngBin,
		},
		testFile{
			p: "assets/css/one.scss",
			sc: `body { background: url("/assets/img/test.png"); }` + "\n" +
				`.test2 { background: url("/assets/img/test.4x.png"); }` + "\n" +
				`.test2 { background: url("/assets/img/test.4x2.png"); }` + "\n" +
				`.test2 { background: url("/assets/img/test.4x2-q81.png"); }` + "\n" +
				`.test { background: url("/assets/img/test.4x2c-q90.png"); }`,
		},
	)
	defer tt.cleanup()

	tt.a.True(fExists(tt.finalPath("public/assets/img/test.png")))
	tt.a.True(fExists(tt.finalPath("public/assets/img/test.4x.png")))
	tt.a.True(fExists(tt.finalPath("public/assets/img/test.4x2.png")))
	tt.a.True(fExists(tt.finalPath("public/assets/img/test.4x2-q81.png")))
	tt.a.True(fExists(tt.finalPath("public/assets/img/test.4x2c-q90.png")))
}

func TestPublicCleanup(t *testing.T) {
	t.Parallel()
	tt := testNew(t, nil,
		testFile{p: "public/test.html"},
		testFile{p: "public/nope.html"},
	)
	defer tt.cleanup()

	tt.a.False(fExists(tt.finalPath("public/test.html")))
	tt.a.False(fExists(tt.finalPath("public/nope.html")))
}
