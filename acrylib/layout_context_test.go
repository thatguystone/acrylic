package acryliclib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutContextSorting(t *testing.T) {
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
	for i, p := range tt.lastSite.lsctx.Pages {
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
			tt.lastSite.lsctx.Pages)
	}
}
