package acrylib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTplPagesSorting(t *testing.T) {
	t.Parallel()

	testFiles := []testFile{
		testFile{p: "content/a/2015-06-07.html"},
		testFile{p: "content/a/2015-06-06.html"},
		testFile{p: "content/a/2015-06-05.html"},
		testFile{p: "content/a/1.html"},
		testFile{p: "content/a/2.html"},
		testFile{p: "content/a/3.html"},
		testFile{p: "content/a/sub/2015-06-07.html"},
		testFile{p: "content/a/sub/2015-06-06.html"},
		testFile{p: "content/a/sub/2015-06-05.html"},
		testFile{p: "content/a/sub/1.html"},
		testFile{p: "content/a/sub/2.html"},
		testFile{p: "content/a/sub/3.html"},
	}

	tt := testNew(t, true, nil, testFiles...)
	defer tt.cleanup()

	sorted := true
	for i, p := range tt.lastSite.tplSite.Pages {
		if filepath.Join(tt.cfg.Root, testFiles[i].p) != p.c.f.srcPath {
			sorted = false
			break
		}
	}

	if !sorted {
		fs := []string{}
		for _, tf := range testFiles {
			fs = append(fs, fChangeExt(fDropFirst(tf.p), ""))
		}

		t.Fatalf("files are not sorted right:\n"+
			"Expected: %s\n"+
			"     Got: %s",
			"["+strings.Join(fs, " ")+"]",
			tt.lastSite.tplSite.Pages)
	}
}

func TestTplMenuBasic(t *testing.T) {
	t.Parallel()

	cfg := testConfig(true)
	tt := testNew(t, true, cfg,
		testFile{
			p: "layouts/_single.html",
			sc: `{% macro dumpMenu(menu) %}` +
				`| {% for m in menu %}` +
				`{% if m.IsChildActive %}Active:{% endif %}` +
				`{{ m.Title }} | ` +
				`{% if m.Childs|length %}({{ dumpMenu(m.Childs) }}) {% endif %}` +
				`{% endfor %}` +
				`{% endmacro %}` +
				`{{ dumpMenu(Site.Menus.main) }}` + "\n",
		},

		testFile{
			p:  "content/string0.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p:  "content/string1.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p:  "content/string2.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p:  "content/string2/page0.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p:  "content/string2/page1.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p:  "content/string2/page2.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p:  "content/string2/page3.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p:  "content/string2/page3/sub0.html",
			sc: "---\nmenu: main\n---",
		},
		testFile{
			p: "content/string2/page3/sub1.html",
		},
		testFile{
			p:  "content/string2/page4.html",
			sc: "---\nmenu: main\n---",
		},

		testFile{
			p: "content/complex0.html",
			sc: "---\nmenu:\n" +
				"  main:\n" +
				"    title: Complex 0\n" +
				"    weight: 50\n" +
				"  foot:\n" +
				"---",
		},

		testFile{
			p: "content/complex1.html",
			sc: "--- json\n" +
				"{\"menu\": {" +
				"\"main\": {\"title\": \"Complex 1\", \"weight\": 100}," +
				"\"foot\": {}}}\n" +
				"---",
		},

		testFile{
			p:  "content/slice0.html",
			sc: "---\nmenu: [main, foot]\n---",
		},
		testFile{
			p:  "content/slice1.html",
			sc: "--- json\n{\"menu\": [\"main\", \"foot\"]}\n---",
		},
	)
	defer tt.cleanup()

	tt.contents("string0.html",
		`| Complex 1 | Complex 0 | Slice0 | Slice1 | Active:String0 | String1 | String2 | (| Page0 | Page1 | Page2 | Page3 | (| Sub0 | ) Page4 | )`)

	tt.contents("string2/page3/sub0.html",
		`| Complex 1 | Complex 0 | Slice0 | Slice1 | String0 | String1 | Active:String2 | (| Page0 | Page1 | Page2 | Active:Page3 | (| Active:Sub0 | ) Page4 | )`)
}
