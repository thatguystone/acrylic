package tmpl

import (
	"bytes"
	"fmt"
	"path/filepath"
	"time"

	"github.com/flosch/pongo2"
	"github.com/thatguystone/acrylic/internal/config"
	"github.com/thatguystone/acrylic/internal/file"
	"github.com/thatguystone/acrylic/internal/imgs"
	"github.com/thatguystone/cog/cfs"
)

type ac struct {
	args Args
	f    file.F
}

type Dims struct {
	LG Dim
	MD Dim
	SM Dim
	XS Dim
}

type Dim struct {
	W, H int
}

func (ac *ac) Cfg() *config.C {
	return ac.args.Cfg
}

func (ac *ac) Data(file string) interface{} {
	return ac.args.Data.Get(file)
}

func (ac *ac) Img(src string) *imgs.Img {
	path := src
	if !filepath.IsAbs(src) {
		path = filepath.Join(ac.f.Src, "../", src)
	} else {
		path = filepath.Join(ac.args.Cfg.ContentDir, src)
	}

	img := ac.args.Imgs.Get(path)

	if img == nil {
		ac.args.Log.Errorf("image not found: %s (resolved to %s)", src, path)
		return nil
	}

	return img
}

func (ac *ac) AllImgs() []string {
	return ac.args.Imgs.All(true)
}

func (ac *ac) Dims(lw, lh, mw, mh, sw, sh, xsw, xsh int) Dims {
	return Dims{
		LG: Dim{lw, lh},
		MD: Dim{mw, mh},
		SM: Dim{sw, sh},
		XS: Dim{xsw, xsh},
	}
}

func (ac *ac) JSTags() *pongo2.Value {
	b := bytes.Buffer{}

	if !ac.args.Cfg.Debug {
		fmt.Fprintf(&b,
			`<script type="text/javascript" src="/%s"></script>`,
			ac.cacheBustAsset("all.js"))
	} else {
		for _, js := range ac.args.Cfg.JS {
			fmt.Fprintf(&b,
				`<script type="text/javascript" src="/%s"></script>`,
				filepath.Join(ac.args.Cfg.AssetsDir, js))
		}
	}

	return pongo2.AsSafeValue(b.String())
}

func (ac *ac) CSSTags() *pongo2.Value {
	b := bytes.Buffer{}

	if !ac.args.Cfg.Debug {
		fmt.Fprintf(&b,
			`<link rel="stylesheet" href="/%s" />`,
			ac.cacheBustAsset("all.css"))
	} else {
		for _, css := range ac.args.Cfg.CSS {
			fmt.Fprintf(&b,
				`<link rel="stylesheet" href="/%s" />`,
				filepath.Join(ac.args.Cfg.AssetsDir, cfs.ChangeExt(css, ".css")))
		}
	}

	return pongo2.AsSafeValue(b.String())
}

func (ac *ac) cacheBustAsset(path string) string {
	path = filepath.Join(ac.args.Cfg.AssetsDir, path)

	if !ac.args.Cfg.CacheBust {
		return path
	}

	return fmt.Sprintf("%s?%d", path, time.Now().Unix())
}
