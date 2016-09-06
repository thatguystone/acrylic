package imgs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/thatguystone/acrylic/internal/afs"
	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/state"
	"github.com/thatguystone/cog/cfs"
	"gopkg.in/yaml.v2"
)

type Img struct {
	file.F
	st        *state.S
	inGallery bool
}

func newImg(st *state.S, f file.F, isContent bool) (*Img, error) {
	metaFile := f.Src + ".meta"
	fm := map[string]interface{}{}
	if exists, _ := cfs.FileExists(metaFile); exists {
		b, err := ioutil.ReadFile(metaFile)
		if err == nil {
			err = yaml.Unmarshal(b, &fm)
		}

		if err != nil {
			return nil, err
		}
	} else if isContent {
		cfs.Write(metaFile, []byte("---\ntitle: \n---\n"))
	}

	title := ""
	if t, ok := fm["title"].(string); ok {
		title = t
	}

	inGallery := isContent
	if g, ok := fm["gallery"].(bool); ok {
		inGallery = g
	}

	f.Title = title
	f.Meta = fm

	img := &Img{
		F:         f,
		st:        st,
		inGallery: inGallery,
	}

	return img, nil
}

func (img *Img) IsGif() bool {
	return img != nil && filepath.Ext(img.Src) == ".gif"
}

func (img *Img) Scale(w, h int, crop bool, quality int) string {
	if img == nil {
		return "<IMAGE NOT FOUND>"
	}

	if quality == 0 {
		quality = 100
	}

	if w == 0 && h == 0 && !crop && quality == 100 {
		img.st.Run.Do(img.copy)
		return img.cacheBustedURL(img.URL)
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

	suffix += filepath.Ext(img.Dst)

	img.st.Run.Do(func() {
		dst := cfs.ChangeExt(img.Dst, suffix)
		if !afs.SrcChanged(img.Src, dst) {
			img.updateDst(dst)
			return
		}

		args := []string{
			img.Src,
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
			img.st.Errs.Errorf(img.Src, "failed to scale: %v", err)
			return
		}

		cmd := exec.Command("convert", args...)

		eb := bytes.Buffer{}
		cmd.Stderr = &eb

		err = cmd.Run()
		if err != nil {
			img.st.Errs.Errorf(img.Src,
				"failed to scale: %v: stderr=%s",
				err,
				eb.String())
			return
		}

		img.updateDst(dst)
	})

	return img.cacheBustedURL(cfs.ChangeExt(img.URL, suffix))
}

func (img *Img) cacheBustedURL(url string) string {
	if !img.st.Cfg.CacheBust {
		return url
	}

	return fmt.Sprintf("%s?%d", url, img.Info.ModTime().Unix())
}

func (img *Img) copy() {
	err := cfs.Copy(img.Src, img.Dst)
	if err != nil {
		img.st.Errs.Errorf(img.Src, "while copying: %v", err)
	} else {
		img.updateDst(img.Dst)
	}
}

func (img *Img) updateDst(dst string) {
	os.Chtimes(dst, img.Info.ModTime(), img.Info.ModTime())
	img.st.Unused.Used(dst)
}
