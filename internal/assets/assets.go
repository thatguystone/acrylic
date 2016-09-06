package assets

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/thatguystone/acrylic/internal/afs"
	"github.com/thatguystone/acrylic/internal/imgs"
	"github.com/thatguystone/acrylic/internal/min"
	"github.com/thatguystone/acrylic/internal/state"
	"github.com/thatguystone/cog/cfs"
)

type A struct {
	st   *state.S
	imgs *imgs.Imgs
}

var (
	reCSSURL    = regexp.MustCompile(`url\("?(.*?)"?\)`)
	reCSSScaled = regexp.MustCompile(`.*(\.((\d*)x(\d*)(c?)(-q(\d*))?)).*`)
)

func New(imgs *imgs.Imgs, st *state.S) *A {
	return &A{
		st:   st,
		imgs: imgs,
	}
}

func (a *A) assetPath(path string) (src, dst string) {
	src = filepath.Join(a.st.Cfg.AssetsDir, path)
	dst = filepath.Join(a.st.Cfg.PublicDir, a.st.Cfg.AssetsDir, path)
	return
}

func (a *A) copy(src string) {
	src, dst := a.assetPath(src)
	afs.CopyState(a.st, src, dst)
}

func (a *A) Render() {
	if a.st.Cfg.Debug {
		a.debug()
	} else {
		a.prod()
	}
}

func (a *A) debug() {
	for _, js := range a.st.Cfg.JS {
		func(js string) {
			a.st.Run.Do(func() {
				a.copy(js)
			})
		}(js)
	}

	for _, css := range a.st.Cfg.CSS {
		func(css string) {
			a.st.Run.Do(func() {
				src, dst := a.assetPath(css)

				if filepath.Ext(css) == ".scss" {
					a.compileScssToFile(src, dst)
				} else {
					a.copy(css)
				}

				a.processCSSAssets(cfs.ChangeExt(dst, ".css"))
			})
		}(css)
	}
}

func (a *A) prod() {
	a.st.Run.Do(a.prodJS)
	a.st.Run.Do(a.prodCSS)
}

func (a *A) prodJS() {
	dstPath, dst, src, cleanup, ok := a.files("all.js", a.st.Cfg.JS, "", nil)
	if !ok {
		return
	}

	defer cleanup()

	var err error
	if len(a.st.Cfg.JSCompiler) > 0 {
		cmd := exec.Command(a.st.Cfg.JSCompiler[0], a.st.Cfg.JSCompiler[1:]...)
		cmd.Stdin = src
		cmd.Stdout = dst

		eb := bytes.Buffer{}
		cmd.Stderr = &eb

		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("%v: %v", err, eb.String())
		}
	} else {
		err = min.Minify("text/javascript", dst, src)
	}

	if err != nil {
		a.st.Errs.Errorf(dstPath, "failed to compress JS: %v", err)
	} else {
		a.st.Unused.Used(dstPath)
	}
}

func (a *A) prodCSS() {
	dstPath, dst, src, cleanup, ok := a.files("all.css", a.st.Cfg.JS, ".scss", a.compileScss)
	if !ok {
		return
	}

	defer cleanup()

	err := min.Minify("text/css", dst, src)
	if err != nil {
		a.st.Errs.Errorf(dstPath, "failed to compress CSS: %v", err)
	} else {
		a.st.Unused.Used(dstPath)
		a.processCSSAssets(dstPath)
	}
}

func (a *A) compileScss(src string, out io.Writer) error {
	args := append([]string{}, a.st.Cfg.SassCompiler...)
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

func (a *A) compileScssToFile(src, dstPath string) {
	var dst io.ReadWriteCloser

	dstPath = cfs.ChangeExt(dstPath, ".css")

	dst, err := cfs.Create(dstPath)
	if err == nil {
		defer dst.Close()
		err = a.compileScss(src, dst)
	}

	if err != nil {
		a.st.Errs.Errorf(src, "failed to compile SCSS: %v", err)
	} else {
		a.st.Unused.Used(dstPath)
	}
}

func (a *A) processCSSAssets(path string) {
	sheet, err := ioutil.ReadFile(path)
	if err != nil {
		a.st.Errs.Errorf(path, "%v", err)
		return
	}

	pfx := filepath.Clean("/" + a.st.Cfg.AssetsDir + "/")

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

THIS IS WRONG
		path := filepath.Join(a.st.Cfg.AssetsDir, url)

		img := a.imgs.Get(path)
		if img == nil {
			a.st.Errs.Errorf(path,
				"image not found: %s", url)
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
		a.st.Errs.Errorf(path, "%v", err)
		return
	}
}

func (a *A) files(dst string, srcs []string, ext string, extCb func(string, io.Writer) error) (
	dstPath string,
	dstW io.Writer,
	srcR io.Reader,
	cleanup func(),
	ok bool) {

	var fs []*os.File
	var readers []io.Reader

	cleanup = func() {
		for _, f := range fs {
			f.Close()
		}
	}

	defer func() {
		if !ok {
			cleanup()
		}
	}()

	for _, src := range srcs {
		src, _ = a.assetPath(src)

		var err error
		if filepath.Ext(src) == ext && extCb != nil {
			b := &bytes.Buffer{}
			readers = append(readers, b)

			err = extCb(src, b)
		} else {
			var f *os.File
			f, err = os.Open(src)
			if err == nil {
				fs = append(fs, f)
				readers = append(readers, f)
			}
		}

		if err != nil {
			a.st.Errs.Errorf(src, "%v", err)
			return
		}
	}

	_, dstPath = a.assetPath(dst)
	f, err := cfs.Create(dstPath)
	if err != nil {
		a.st.Errs.Errorf(dstPath, "%v", err)
		return
	}

	fs = append(fs, f)

	dstW = f
	srcR = io.MultiReader(readers...)
	ok = true

	return
}
