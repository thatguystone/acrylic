package acrylic

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/goji/param"
	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/stringc"
)

// Image implements a caching image scaler
type Image struct {
	Root  string // Directory to search for the original image
	Cache string // Where to cache scaled images
	bg    bgWork
}

func (im *Image) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	src := path.Join(im.Root, r.URL.Path)

	info, ok := im.stat(w, src)
	switch {
	case !ok:

	case info == nil:
		im.serveCache(w, r)

	default:
		im.serveScale(w, r, src, info)
	}
}

func (im *Image) stat(w http.ResponseWriter, path string) (os.FileInfo, bool) {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return nil, true
	}

	if err != nil {
		log.Printf("[img] E: failed to stat %s: %v", path, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, false
	}

	return info, true
}

func (im *Image) cachePath(r *http.Request) string {
	return path.Join(im.Cache, r.URL.Path)
}

func (im *Image) serveCache(w http.ResponseWriter, r *http.Request) {
	cachePath := im.cachePath(r)

	err := im.bg.wait(cachePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	info, ok := im.stat(w, cachePath)
	switch {
	case !ok:

	case info == nil:
		http.NotFound(w, r)

	case strings.Contains(r.Header.Get("Accept"), Accept):
		Path(w, "application/octet-stream", cachePath)

	default:
		// Provides Last-Modified caching
		http.ServeFile(w, r, cachePath)
	}
}

func (im *Image) serveScale(
	w http.ResponseWriter, r *http.Request,
	src string, srcInfo os.FileInfo) {

	r.ParseForm()

	img, err := newImg(r.Form)
	if err != nil {
		log.Printf("[img] E: invalid args for %s: %v", src, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dstBase, err := img.cacheName(src)
	if err != nil {
		log.Printf("[img] E: cache key gen for %s failed: %v", src, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dst := path.Join(im.Cache, path.Dir(r.URL.Path), dstBase)
	cacheInfo, ok := im.stat(w, dst)
	if !ok {
		return
	}

	if cacheInfo == nil || !srcInfo.ModTime().Equal(cacheInfo.ModTime()) {
		im.bg.do(dst, func() error {
			err := img.scale(src, dst)
			if err != nil {
				log.Printf("[img] E: failed to scale %s: %v", src, err)
				return err
			}

			err = os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())
			if err != nil {
				log.Printf("[img] E: failed to set modtime for %s: %v", dst, err)
				return err
			}

			return nil
		})
	}

	http.Redirect(w, r, dstBase, http.StatusFound)
}

type img struct {
	W, H int    // Dimensions
	D    int    // Density (for HiDPI devices)
	Q    int    // Image quality
	Crop bool   // Crop the image?
	Ext  string // Convert to this format
}

func newImg(form url.Values) (img img, err error) {
	err = param.Parse(form, &img)
	if err != nil {
		return
	}

	if img.D == 0 {
		img.D = 1
	}

	img.W *= img.D
	img.H *= img.D

	if img.Q == 0 {
		img.Q = 100
	}

	if img.W == 0 && img.H == 0 {
		img.Crop = false
	}

	if img.Ext != "" {
		img.Ext = "." + strings.Trim(img.Ext, ".")
	}

	return
}

func (img img) cacheName(src string) (string, error) {
	buff := new(bytes.Buffer)

	dims := img.getDims()
	if dims != "" {
		buff.WriteString(dims)
	}

	if img.Q != 100 {
		fmt.Fprintf(buff, "q%d", img.Q)
	}

	if img.Crop {
		fmt.Fprint(buff, "c")
	}

	if buff.Len() > 0 {
		buff.WriteByte('-')
	}

	var fingerp string
	f, err := os.Open(src)
	if err == nil {
		fingerp, err = shortFingerprint(f)
		f.Close()
	}

	if err != nil {
		return "", err
	}

	buff.WriteString(fingerp)

	ext := img.Ext
	if ext == "" {
		ext = path.Ext(src)
	}

	buff.WriteString(ext)

	return cfs.ChangeExt(path.Base(src), buff.String()), nil
}

func (img img) getDims() string {
	buff := new(bytes.Buffer)

	if img.W != 0 {
		fmt.Fprintf(buff, "%d", img.W)
	}

	buff.WriteByte('x')

	if img.H != 0 {
		fmt.Fprintf(buff, "%d", img.H)
	}

	if buff.Len() == 1 {
		buff.Reset()
	}

	return buff.String()
}

func (img img) scale(src, dst string) error {
	var args []string

	dims := img.getDims()
	if dims != "" {
		if img.Crop {
			dims += "^"

			args = append(args,
				"-gravity", "center",
				"-extent", dims)
		}

		args = append(args,
			"-scale", dims)
	}

	if img.Q != 100 {
		args = append(args,
			"-quality", fmt.Sprintf("%d", img.Q))
	}

	err := cfs.CreateParents(dst)
	if err != nil {
		return err
	}

	if len(args) == 0 && path.Ext(src) == path.Ext(dst) {
		return cfs.Copy(src, dst)
	}

	args = append(args, src, dst)
	out, err := exec.Command("convert", args...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("convert: %v\n%s",
			err.Error(),
			stringc.Indent(string(out), indent))
	}

	return err
}
