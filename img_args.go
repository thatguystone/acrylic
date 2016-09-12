package acrylic

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/thatguystone/cog/cfs"
)

type imgArgsNS struct{}

var imgArgs = imgArgsNS{}

func (ia imgArgsNS) parseName(im *img, vals string) (err error) {
	dstExt := filepath.Ext(vals)
	if dstExt != "" {
		vals = cfs.DropExt(vals)
	}

	form, err := url.ParseQuery(vals)
	if err != nil {
		return
	}

	if dstExt != "" {
		form.Set("dstExt", dstExt)
	}

	_, err = ia.parseForm(im, form)

	return
}

func (ia imgArgsNS) parseForm(
	im *img,
	form url.Values) (usedAny bool, err error) {

	for k := range form {
		used, err := ia.parseOne(im, k, form.Get(k))
		if err != nil {
			return false, err
		}

		usedAny = usedAny || used
	}

	return
}

func (ia imgArgsNS) parseOne(im *img, k, v string) (used bool, err error) {
	// An empty arg is stupid, but not an error. Just ignore it.
	if v == "" {
		return
	}

	used = true
	switch k {
	case "w":
		im.oW, err = ia.intArg(v)

	case "h":
		im.oH, err = ia.intArg(v)

	case "c":
		im.crop = true

	case "q":
		im.quality, err = ia.intArg(v)
		if err == nil && im.quality == 0 {
			im.quality = 100
		}

	case "d":
		im.density, err = ia.intArg(v)
		if err == nil && im.density == 0 {
			im.density = 1
		}

	case "dstExt":
		im.dstExt = "." + strings.Trim(v, ".")

	default:
		used = false
	}

	return
}

// To be called after _all_ parsing operations are complete.
func (ia imgArgsNS) postParse(im *img) {
	im.w = im.oW * im.density
	im.h = im.oH * im.density

	if im.w == 0 && im.h == 0 {
		im.crop = false
	}

	if im.dstExt == filepath.Ext(im.srcPath) {
		im.dstExt = ""
	}
}

func (ia imgArgsNS) format(im *img) (args string) {
	var s []string

	add := func(k, v string) {
		s = append(s, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}

	if im.w != 0 {
		add("w", strconv.Itoa(im.w))
	}

	if im.h != 0 {
		add("h", strconv.Itoa(im.h))
	}

	if im.crop {
		add("c", "t")
	}

	if im.quality != 100 {
		add("q", strconv.Itoa(im.quality))
	}

	args = strings.Join(s, "&")

	if args != "" || im.dstExt != "" {
		ext := im.dstExt
		if ext == "" {
			ext = filepath.Ext(im.srcPath)
		}

		args += ext
	}

	return
}

func (ia imgArgsNS) cmdArgs(im *img) (args []string) {
	dims := ""

	// Use something like '400x' to scale to a width of 400
	if im.w != 0 {
		dims += strconv.Itoa(im.w)
	}

	dims += "x"

	if im.h != 0 {
		dims += strconv.Itoa(im.h)
	}

	if dims != "x" {
		scaleDims := dims
		if im.crop {
			scaleDims += "^"

			args = append(args,
				"-gravity", "center",
				"-extent", dims)
		}

		args = append(args,
			"-scale", scaleDims)
	}

	if im.quality != 100 {
		args = append(args,
			"-quality", fmt.Sprintf("%d", im.quality))
	}

	return
}

func (ia imgArgsNS) intArg(val string) (int, error) {
	i, err := strconv.Atoi(val)
	if err == nil && i < 0 {
		err = fmt.Errorf("arg must be > 0")
	}

	return i, err
}
