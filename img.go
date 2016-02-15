package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/thatguystone/acrylic/internal/afs"
	"github.com/thatguystone/cog/cfs"
)

type images struct {
	all   []*image
	imgs  map[string]*image
	byCat map[string][]*image
}

type image struct {
	ss        *siteState
	src       string
	dst       string
	sortName  string
	info      os.FileInfo
	Cat       string
	Title     string
	Date      time.Time
	url       string
	Meta      map[string]interface{}
	inGallery bool
}

type imageSlice []*image

func (imgs *images) add(img *image) {
	cat := imgs.byCat[img.Cat]
	cat = append(cat, img)
	imgs.byCat[img.Cat] = cat

	imgs.imgs[img.src] = img

	imgs.all = append(imgs.all, img)
}

func (imgs *images) sort() {
	sort.Sort(imageSlice(imgs.all))

	for _, ims := range imgs.byCat {
		sort.Sort(imageSlice(ims))
	}
}

func (imgs *images) get(path string) *image {
	return imgs.imgs[path]
}

func (img *image) IsGif() bool {
	return img != nil && filepath.Ext(img.src) == ".gif"
}

func (img *image) Scale(w, h int, crop bool, quality int) string {
	if img == nil {
		return "<IMAGE NOT FOUND>"
	}

	if quality == 0 {
		quality = 100
	}

	if w == 0 && h == 0 && !crop && quality == 100 {
		img.ss.pool.Do(img.copy)
		return img.cacheBustedURL(img.url)
	}

	dims := ""

	// Use something like '400x' to scale to a width of 400
	if w != 0 {
		dims += fmt.Sprintf("%d", w)
	}

	dims += "x"

	if h != 0 {
		dims += fmt.Sprintf("%d", h)
	}

	suffix := dims
	scaleDims := dims

	if crop {
		suffix += "c"
		scaleDims += "^"
	}

	if quality != 100 {
		suffix += fmt.Sprintf("-q%d", quality)
	}

	suffix += filepath.Ext(img.dst)

	img.ss.pool.Do(func() {
		dst := cfs.ChangeExt(img.dst, suffix)
		if !afs.SrcChanged(img.src, dst) {
			img.updateDst(dst)
			return
		}

		args := []string{
			img.src,
			"-quality", fmt.Sprintf("%d", quality),
			"-scale", scaleDims,
		}

		if crop {
			args = append(args,
				"-gravity", "center",
				"-extent", dims)
		}

		args = append(args, dst)
		err := cfs.CreateParents(dst)
		if err != nil {
			img.ss.errs.add(img.src, fmt.Errorf("failed to scale: %v", err))
			return
		}

		cmd := exec.Command("convert", args...)

		eb := bytes.Buffer{}
		cmd.Stderr = &eb

		err = cmd.Run()
		if err != nil {
			img.ss.errs.add(img.src,
				fmt.Errorf("failed to scale: %v: stderr=%s",
					err,
					eb.String()))
			return
		}

		img.updateDst(dst)
	})

	return img.cacheBustedURL(cfs.ChangeExt(img.url, suffix))
}

func (img *image) cacheBustedURL(url string) string {
	if !img.ss.cfg.CacheBust {
		return url
	}

	return fmt.Sprintf("%s?%d", url, img.info.ModTime().Unix())
}

func (img *image) copy() {
	err := cfs.Copy(img.src, img.dst)
	if err != nil {
		img.ss.errs.add(img.src, fmt.Errorf("while copying: %v", err))
	} else {
		img.updateDst(img.dst)
	}
}

func (img *image) updateDst(dst string) {
	os.Chtimes(dst, img.info.ModTime(), img.info.ModTime())
	img.ss.markUsed(dst)
}

func (is imageSlice) Len() int           { return len(is) }
func (is imageSlice) Swap(i, j int)      { is[i], is[j] = is[j], is[i] }
func (is imageSlice) Less(i, j int) bool { return is[i].sortName > is[j].sortName }
