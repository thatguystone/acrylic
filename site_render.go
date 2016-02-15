package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/thatguystone/cog/cfs"
)

const (
	scissors     = `<!-- >8 acrylic-content -->`
	scissorsEnd  = `<!-- acrylic-content 8< -->`
	moreScissors = `<!--more-->`
)

var (
	reCSSURL    = regexp.MustCompile(`url\("?(.*?)"?\)`)
	reCSSScaled = regexp.MustCompile(`.*(\.((\d*)x(\d*)(c?)(-q(\d*))?)).*`)
)

func (ss *siteState) renderPages() {
	for _, pgs := range ss.pages.byCat {
		for _, pg := range pgs {
			if pg.isListPage {
				continue
			}

			func(pg *page) {
				ss.pool.Do(func() {
					ss.renderPage(pg)
				})
			}(pg)
		}
	}
}

func (ss *siteState) renderPage(pg *page) {
	tmpl, err := ss.tmplSet.FromString(pg.Content)
	if err != nil {
		ss.errs.add(pg.src, err)
		return
	}

	content, err := tmpl.Execute(ss.tmplVars(pg))
	if err != nil {
		ss.errs.add(pg.src, err)
		return
	}

	start := strings.Index(content, scissors)
	end := strings.Index(content, scissorsEnd)

	pg.Content = ""
	if start >= 0 && end >= 0 {
		pg.Content = content[start+len(scissors) : end]

		end := strings.Index(pg.Content, moreScissors)
		if end >= 0 {
			pg.Summary = pg.Content[:end]
		}
	}

	ss.writePage(pg, pg.dst, content)
}

func (ss *siteState) renderListPages() {
	for _, pgs := range ss.pages.byCat {
		for _, pg := range pgs {
			if !pg.isListPage {
				continue
			}

			func(pg *page) {
				ss.pool.Do(func() {
					ss.renderListPage(pg)
				})
			}(pg)
		}
	}
}

func (ss *siteState) renderListPage(pg *page) {
	tmpl, err := ss.tmplSet.FromString(pg.Content)
	if err != nil {
		ss.errs.add(pg.src, err)
		return
	}

	pages := ss.pages.posts(pg.Cat)

	total := len(pages)
	pageCount := int(math.Ceil(float64(total) / float64(ss.cfg.PerPage)))

	for i := 0; i < pageCount; i++ {
		listStart := i * ss.cfg.PerPage
		listEnd := listStart + ss.cfg.PerPage

		if listEnd > total {
			listEnd = total
		}

		content, err := tmpl.Execute(ss.tmplVars(pg).Update(
			pongo2.Context{
				"page":        pg,
				"pages":       pages[listStart:listEnd],
				"pageNum":     i + 1,
				"listHasNext": i < (pageCount - 1),
			}))
		if err != nil {
			ss.errs.add(pg.src, err)
			return
		}

		dst := pg.dst
		if i > 0 {
			dst = filepath.Join(
				ss.baseDir,
				ss.cfg.PublicDir,
				filepath.Dir(pg.URL),
				"page", fmt.Sprintf("%d", i+1),
				"index.html")
		}

		ss.writePage(pg, dst, content)
	}
}

func (ss *siteState) renderAssets() {
	assetPath := func(path string) (src, dst string) {
		src = filepath.Join(ss.baseDir, ss.cfg.AssetsDir, path)
		dst = filepath.Join(ss.baseDir, ss.cfg.PublicDir, ss.cfg.AssetsDir, path)
		return
	}

	rawCopy := func(src string) {
		ss.copyFile(assetPath(src))
	}

	if ss.cfg.Debug {
		for _, js := range ss.cfg.JS {
			func(js string) {
				ss.pool.Do(func() {
					rawCopy(js)
				})
			}(js)
		}

		for _, css := range ss.cfg.CSS {
			func(css string) {
				ss.pool.Do(func() {
					src, dst := assetPath(css)

					if filepath.Ext(css) == ".scss" {
						ss.compileScssToFile(src, dst)
					} else {
						rawCopy(css)
					}

					ss.processCSSAssets(cfs.ChangeExt(dst, ".css"))
				})
			}(css)
		}

		return
	}

	ss.pool.Do(func() {
		b := bytes.Buffer{}
		for _, js := range ss.cfg.JS {
			src, _ := assetPath(js)

			f, err := os.Open(src)
			if err != nil {
				ss.errs.add(src, err)
				return
			}

			_, err = b.ReadFrom(f)
			f.Close()

			if err != nil {
				ss.errs.add(src, err)
				return
			}
		}

		_, dst := assetPath("all.js")
		f, err := cfs.Create(dst)
		if err != nil {
			ss.errs.add(dst, err)
			return
		}

		defer f.Close()

		if len(ss.cfg.JSCompiler) > 0 {
			cmd := exec.Command(ss.cfg.JSCompiler[0], ss.cfg.JSCompiler[1:]...)
			cmd.Stdin = &b
			cmd.Stdout = f

			eb := bytes.Buffer{}
			cmd.Stderr = &eb

			err = cmd.Run()
			if err != nil {
				err = fmt.Errorf("%v: %v", err, eb.String())
			}
		} else {
			err = ss.min.Minify("text/javascript", f, &b)
		}

		if err != nil {
			ss.errs.add(dst, err)
		} else {
			ss.markUsed(dst)
		}
	})

	ss.pool.Do(func() {
		b := bytes.Buffer{}
		for _, css := range ss.cfg.CSS {
			src, _ := assetPath(css)

			var err error
			if filepath.Ext(src) == ".scss" {
				err = ss.compileScss(src, &b)
			} else {
				f, err := os.Open(src)
				if err != nil {
					ss.errs.add(src, err)
					return
				}

				_, err = b.ReadFrom(f)
				f.Close()
			}

			if err != nil {
				ss.errs.add(src, err)
				return
			}
		}

		_, dst := assetPath("all.css")
		f, err := cfs.Create(dst)
		if err != nil {
			ss.errs.add(dst, err)
			return
		}

		err = ss.min.Minify("text/css", f, &b)
		f.Close()

		if err != nil {
			ss.errs.add(dst, err)
		} else {
			ss.markUsed(dst)
			ss.processCSSAssets(dst)
		}
	})
}

func (ss *siteState) compileScss(src string, out io.Writer) error {
	args := append([]string{}, ss.cfg.SassCompiler...)
	args = append(args, src)

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = out

	eb := bytes.Buffer{}
	cmd.Stderr = &eb

	err := cmd.Run()
	if err != nil || eb.Len() > 0 {
		return fmt.Errorf("execute failed: %v, stderr=%s", err, eb.String())
	}

	return nil
}

func (ss *siteState) compileScssToFile(src, dstPath string) {
	var dst io.ReadWriteCloser

	dstPath = cfs.ChangeExt(dstPath, ".css")

	dst, err := cfs.Create(dstPath)
	if err == nil {
		defer dst.Close()
		err = ss.compileScss(src, dst)
	}

	if err != nil {
		ss.errs.add(src, err)
	} else {
		ss.markUsed(dstPath)
	}
}

func (ss *siteState) processCSSAssets(path string) {
	sheet, err := ioutil.ReadFile(path)
	if err != nil {
		ss.errs.add(path, err)
		return
	}

	pfx := filepath.Clean("/" + ss.cfg.AssetsDir + "/")

	matches := reCSSURL.FindAllSubmatch(sheet, -1)
	for _, match := range matches {
		absURL := string(match[1])

		if !strings.HasPrefix(absURL, pfx) {
			continue
		}

		url := absURL
		w := 0
		h := 0
		crop := false
		quality := 100

		parts := reCSSScaled.FindStringSubmatch(url)
		if len(parts) > 0 {
			w, _ = strconv.Atoi(parts[3])
			h, _ = strconv.Atoi(parts[4])
			crop = len(parts[5]) > 0
			quality, _ = strconv.Atoi(parts[7])

			url = strings.Replace(url, parts[1], "", -1)
		}

		path := filepath.Join(ss.baseDir, url)

		img := ss.imgs.get(path)
		if img == nil {
			ss.errs.add(path,
				fmt.Errorf("image not found: %s (resolved to %s)", url, path))
			return
		}

		final := img.Scale(w, h, crop, quality)
		sheet = bytes.Replace(
			sheet,
			match[0],
			[]byte(fmt.Sprintf(`url("%s")`, final)),
			-1)
	}

	err = ioutil.WriteFile(path, sheet, 0640)
	if err != nil {
		ss.errs.add(path, err)
		return
	}
}

func (ss *siteState) copyBlobs() {
	for _, b := range ss.blobs {
		func(b *blob) {
			ss.pool.Do(func() {
				ss.copyFile(b.src, b.dst)
			})
		}(b)
	}
}

func (ss *siteState) writePage(pg *page, dst, content string) {
	ss.markUsed(dst)

	if !ss.cfg.Debug {
		b := bytes.Buffer{}
		err := ss.min.Minify("text/html", &b, strings.NewReader(content))
		if err != nil {
			ss.errs.add(pg.src, err)
			return
		}

		content = b.String()
	}

	err := cfs.Write(dst, []byte(content))
	if err != nil {
		ss.errs.add(pg.src, err)
		return
	}
}

func (ss *siteState) tmplVars(pg *page) pongo2.Context {
	return pongo2.Context{
		"ac":   newTmplVars(ss, pg),
		"page": pg,
	}
}

func (ss *siteState) copyFile(src, dst string) {
	err := cfs.Copy(src, dst)
	if err != nil {
		ss.errs.add(src, fmt.Errorf("failed to copy: %v", err))
		return
	}

	info, err := os.Stat(src)
	if err != nil {
		ss.errs.add(src, err)
		return
	}

	os.Chtimes(dst, info.ModTime(), info.ModTime())
	ss.markUsed(dst)
}
