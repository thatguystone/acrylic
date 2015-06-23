package acrylib

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

type contentGenImg struct {
	s *site
	c *content

	mtx    sync.Mutex
	scaled map[string]image.Point
}

type img struct {
	src  string
	ext  string
	w    int
	h    int
	crop imgCrop
}

type imgCrop int

const (
	cropNone imgCrop = iota
	cropLeft
	cropCentered
	cropLen
)

const imgUniquifier = "img"

var (
	imgExts      = map[string]bool{}
	imgExtsSlice = []string{
		".bmp",
		".gif",
		".jpeg",
		".jpg",
		".png",
		".tiff",

		// TODO(astone): look into checking for imagemagick/graphics magick to get thumbs of PFDs and stuff?
	}
)

func init() {
	for _, ext := range imgExtsSlice {
		imgExts[ext] = true
	}
}

func getContentImgGener(s *site, c *content, ext string) (contentGener, contentType) {
	if !imgExts[ext] {
		return nil, contInvalid
	}

	gi := &contentGenImg{
		s:      s,
		c:      c,
		scaled: map[string]image.Point{},
	}

	return gi, contImg
}

func (gi *contentGenImg) finalExt(c *content) string {
	return ".html"
}

func (gi *contentGenImg) render(s *site, c *content) (content []byte, err error) {
	panic("not yet implemented")
}

func (gi *contentGenImg) generate(content []byte, dstPath string, s *site, c *content) (
	wroteOwnFile bool,
	err error) {

	panic("not yet implemented")

	// TODO(astone): generate image pages (in dir `imgUniquifier`)
	// TODO(astone): be sure to include asset trailers
}

func (gi *contentGenImg) scale(img img) (w, h int, dstPath string, err error) {
	ext := gi.getNewExt(img)
	dstPath, alreadyClaimed, err := gi.c.claimOtherExt(ext)
	if err != nil {
		return
	}

	if alreadyClaimed {
		for {
			gi.mtx.Lock()
			p, ok := gi.scaled[dstPath]
			gi.mtx.Unlock()

			if ok {
				w, h = p.X, p.Y
				return
			}

			time.Sleep(time.Millisecond)
		}
	}

	w, h = img.w, img.h
	defer func() {
		gi.mtx.Lock()
		gi.scaled[dstPath] = image.Point{X: w, Y: h}
		gi.mtx.Unlock()
	}()

	if !fSrcChanged(gi.c.f.srcPath, dstPath) {
		var f *os.File
		f, err = os.Open(dstPath)
		if err != nil {
			return
		}

		defer f.Close()

		var ig image.Image
		ig, err = imaging.Decode(f)
		if err != nil {
			return
		}

		bounds := ig.Bounds()
		w = bounds.Dx()
		h = bounds.Dy()
		return
	}

	gi.s.stats.addImg()

	f, err := os.Open(gi.c.f.srcPath)
	if err != nil {
		return
	}

	defer f.Close()

	ig, err := imaging.Decode(f)
	if err != nil {
		return
	}

	switch {
	case img.w == 0 && img.h == 0:
		// No resizing

	case img.w != 0 && img.h != 0 && img.crop != cropNone:
		// It doesn't make sense to crop if full dimensions aren't given since
		// it's just scaling if a dimension is missing.
		ig = gi.thumbnailImage(ig, img)

	default:
		ig = gi.resizeImage(ig, img)
	}

	bounds := ig.Bounds()
	w = bounds.Dx()
	h = bounds.Dy()

	err = gi.saveImage(ig, dstPath)
	return
}

func (gi *contentGenImg) thumbnailImage(ig image.Image, img img) image.Image {
	igb := ig.Bounds()
	srcW, srcH := igb.Dx(), igb.Dy()

	scaleW, scaleH := srcW, srcH

	if scaleW < img.w {
		scaleH = (scaleH * img.w) / scaleW
		scaleW = img.w
	}

	if scaleH < img.h {
		scaleW = (scaleW * img.h) / scaleH
		scaleH = img.h
	}

	if scaleW == 0 {
		scaleW = 1
	}

	if scaleH == 0 {
		scaleH = 1
	}

	ig = imaging.Resize(ig, scaleW, scaleH, imaging.Lanczos)

	crop := image.Rectangle{}
	switch img.crop {
	case cropLeft:
		crop = image.Rectangle{
			Min: image.Point{X: 0, Y: 0},
			Max: image.Point{X: img.w, Y: img.h},
		}

	case cropCentered:
		centerX := scaleW / 2
		centerY := scaleH / 2

		x0 := centerX - img.w/2
		y0 := centerY - img.h/2
		x1 := x0 + img.w
		y1 := y0 + img.h

		crop = image.Rectangle{
			Min: image.Point{X: x0, Y: y0},
			Max: image.Point{X: x1, Y: y1},
		}

	default:
		panic(fmt.Errorf("unsupported crop option: %d", img.crop))
	}

	ig = imaging.Crop(ig, crop)

	return ig
}

func (*contentGenImg) resizeImage(ig image.Image, img img) image.Image {
	igb := ig.Bounds()
	srcW, srcH := igb.Dx(), igb.Dy()

	scaleW, scaleH := srcW, srcH

	doScale := img.h == 0 ||
		(scaleW > img.w && img.w != 0) ||
		(scaleW < img.w && scaleH < img.h)
	if doScale {
		scaleH = (scaleH * img.w) / scaleW
		scaleW = img.w
	}

	doScale = img.w == 0 ||
		(scaleH > img.h && img.h != 0) ||
		(scaleW < img.w && scaleH < img.h)
	if doScale {
		scaleW = (scaleW * img.h) / scaleH
		scaleH = img.h
	}

	if scaleW == 0 {
		scaleW = 1
	}

	if scaleH == 0 {
		scaleH = 1
	}

	return imaging.Resize(ig, scaleW, scaleH, imaging.Lanczos)
}

func (gi *contentGenImg) saveImage(ig image.Image, dst string) error {
	f, err := gi.s.fCreate(dst)
	if err != nil {
		return err
	}

	defer f.Close()

	ext := filepath.Ext(dst)
	switch ext {
	case ".bmp":
		return bmp.Encode(f, ig)

	case ".gif":
		opts := &gif.Options{
			NumColors: 256,
		}

		return gif.Encode(f, ig, opts)

	case ".jpeg", ".jpg":
		opts := &jpeg.Options{
			Quality: 95,
		}

		if nrgba, ok := ig.(*image.NRGBA); ok && nrgba.Opaque() {
			rgba := &image.RGBA{
				Pix:    nrgba.Pix,
				Stride: nrgba.Stride,
				Rect:   nrgba.Rect,
			}

			return jpeg.Encode(f, rgba, opts)
		}

		return jpeg.Encode(f, ig, opts)

	case ".png":
		enc := png.Encoder{
			CompressionLevel: png.BestCompression,
		}

		return enc.Encode(f, ig)

	case ".tiff":
		opts := &tiff.Options{
			Compression: tiff.Deflate,
			Predictor:   true,
		}

		return tiff.Encode(f, ig, opts)

	default:
		panic(fmt.Errorf("image type %s is not supported and slipped through", ext))
	}
}

func (*contentGenImg) getNewExt(img img) string {
	ext := ""

	if img.w != 0 || img.h != 0 {
		ext += fmt.Sprintf("%dx%d", img.w, img.h)
	}

	if img.crop != cropNone {
		ext += fmt.Sprintf(".c%c", img.crop.String()[0])
	}

	ext += img.ext

	return ext
}

func (c imgCrop) String() string {
	switch c {
	case cropLeft:
		return "left"
	case cropCentered:
		return "centered"
	default:
		return "none"
	}
}
