package acrylic

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/stringc"
)

type imgHandler struct {
	handler
	s          *Site
	root       string
	fileServer http.Handler
}

func newImgHandler(s *Site, root string) imgHandler {
	return imgHandler{
		s:    s,
		root: root,
		fileServer: http.FileServer(staticDirs{
			root,
			s.Output,
		}),
	}
}

func (h imgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid query: %v", err)
		return
	}

	// Cache buster is useless to us
	r.Form.Del(cacheBuster)

	srcPath := filepath.Join(h.root, upath)

	img, err := newImg(srcPath, r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "invalid parameter: %v", err)
		return
	}

	if !img.needsScale {
		// Provides Last-Modified caching
		h.fileServer.ServeHTTP(w, r)
	} else {
		h.scale(w, r, srcPath, img)
	}
}

func (h imgHandler) scale(
	w http.ResponseWriter, r *http.Request,
	srcPath string, img img) {

	scaledName := img.scaledName()
	dstPath := filepath.Join(h.s.Output,
		filepath.Dir(r.URL.String()),
		scaledName)

	srcStat, ok, _ := h.statFile(srcPath, w, true)
	if !ok {
		return
	}

	dstStat, ok, err := h.statFile(dstPath, w, false)
	if err != nil {
		return
	}

	// If the internal cache is outdated, update before serving
	if !ok || !srcStat.ModTime().Equal(dstStat.ModTime()) {
		err := img.scale(dstPath)
		if err != nil {
			h.errorf(w, err, "failed to scale img")
			return
		}
	}

	http.Redirect(w, r, "./"+scaledName, http.StatusMovedPermanently)
}

type img struct {
	srcPath    string
	suffix     string
	needsScale bool
	args       []string

	w, h    int // If both == 0, just means original size
	crop    bool
	quality int // 0 == original quality
	density int
	dstExt  string
}

func newImg(srcPath string, qs url.Values) (img, error) {
	img := img{
		srcPath: srcPath,
		args:    []string{srcPath},
	}

	err := img.parseQS(qs)
	if err == nil {
		img.setupParams()
	}

	img.needsScale = img.w != 0 || img.h != 0 ||
		img.crop ||
		img.quality != 0 ||
		img.density > 1

	return img, err
}

func (img img) scaledName() string {
	base := filepath.Base(img.srcPath)
	return cfs.ChangeExt(base, img.suffix)
}

func (img *img) parseQS(qs url.Values) (err error) {
	img.w, err = img.parseInt(qs, "w")

	if err == nil {
		img.h, err = img.parseInt(qs, "h")
	}

	if err == nil {
		img.quality, err = img.parseInt(qs, "q")
	}

	if err == nil {
		img.density, err = img.parseInt(qs, "d")
	}

	if img.density == 0 {
		img.density = 1
	}

	img.w *= img.density
	img.h *= img.density

	img.crop = qs.Get("c") != ""
	img.dstExt = qs.Get("ext")

	return err
}

func (img) parseInt(qs url.Values, key string) (int, error) {
	val := qs.Get(key)

	// Empty? That's not an error
	if val == "" {
		return 0, nil
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		err = fmt.Errorf("invalid %s: %v", key, err)
	}

	return i, err
}

func (img *img) setupParams() {
	dims := ""

	// Use something like '400x' to scale to a width of 400
	if img.w != 0 {
		dims += fmt.Sprintf("%d", img.w)
	}

	dims += "x"

	if img.h != 0 {
		dims += fmt.Sprintf("%d", img.h)
	}

	suffix := dims
	scaleDims := dims

	if img.crop {
		suffix += "c"
		scaleDims += "^"

		img.args = append(img.args,
			"-gravity", "center",
			"-extent", dims)
	}

	if dims != "x" {
		img.args = append(img.args,
			"-scale", scaleDims)
	}

	if img.quality != 0 {
		suffix += fmt.Sprintf("-q%d", img.quality)
		img.args = append(img.args,
			"-quality", fmt.Sprintf("%d", img.quality))
	}

	if img.dstExt == "" {
		img.dstExt = filepath.Ext(img.srcPath)
	} else if img.dstExt[0] != '.' {
		img.dstExt = "." + img.dstExt
	}

	img.suffix = suffix + img.dstExt
}

func (img *img) scale(dstPath string) error {
	err := cfs.CreateParents(dstPath)

	if err == nil {
		args := append(img.args, dstPath)

		cmd := exec.Command("convert", args...)
		cmd.Stdout = os.Stdout

		b := bytes.Buffer{}
		cmd.Stderr = &b

		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("%v\n%s", err, stringc.Indent(b.String(), "    "))
		}
	}

	if err != nil {
		err = fmt.Errorf("failed to scale %s: %v", img.srcPath, err)
	}

	return err
}
