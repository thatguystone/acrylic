package acrylib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTplSorting(t *testing.T) {
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

	cfg := testConfig()
	cfg.MinifyHTML = false

	tt := testNew(t, true, cfg,
		testFile{
			p: "layouts/_single.html",
			sc: `main: {% for m in Site.Menus.Get("main").Links %}` +
				`{% if m.SubActive %}Active:{% endif %}` +
				`{{ m.Title }} | ` +
				`{% endfor %}` + "\n" +

				`main.sub: {% for m in Site.Menus.Get("main.sub").Links %}` +
				`{% if m.SubActive %}Active:{% endif %}` +
				`{{ m.Title }} | ` +
				`{% endfor %}` + "\n" +

				`main.sub.sub: {% for m in Site.Menus.Get("main.sub.sub").Links %}` +
				`{% if m.SubActive %}Active:{% endif %}` +
				`{{ m.Title }} | ` +
				`{% endfor %}`,
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
			p:  "content/string3.html",
			sc: "---\nmenu: main.sub\n---",
		},
		testFile{
			p:  "content/string4.html",
			sc: "---\nmenu: main.sub\n---",
		},
		testFile{
			p:  "content/string5.html",
			sc: "---\nmenu: main.sub.\n---",
		},
		testFile{
			p:  "content/string6.html",
			sc: "---\nmenu: main.sub.sub\n---",
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

	// FORCE MENU HEIRARCY BASED ON CONTENT HEIRARCHY: DONT ALLOW NAMED MENUS, JUST NAME THEM AFTER THEIR DIR

	// tt.contents("public/string0.html",
	// 	`| Complex 1 | Complex 0 | Slice0 | Slice1 | Active:String0 | String1 | String2 | String6 |`)
	// tt.contents("public/string5.html",
	// 	``)
}
