package acrylic

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

type imgArgsNS struct{}

var imgArgs = imgArgsNS{}

func (ia imgArgsNS) parseNameArgs(im *img, trailer string) (err error) {
	args := strings.Split(trailer, ".")
	if len(args) == 1 && args[0] == "" {
		return fmt.Errorf("missing name args")
	}

	// If the last arg doesn't look like a k/v pair, it's probably the new img
	// extension
	n := len(args) - 1
	last := args[n]
	if !strings.Contains(last, "=") {
		args[n] = fmt.Sprintf("dstExt=%s", last)
	}

	for _, arg := range args {
		if arg == "" {
			continue
		}

		parts := strings.Split(arg, "=")
		if len(parts) != 2 {
			err = fmt.Errorf("invalid arg: %s", arg)
			return
		}

		_, err = ia.parseOne(im, parts[0], parts[1])
		if err != nil {
			return
		}
	}

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
		im.dstExt = strings.Trim(v, ".")

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

	ext := "." + im.dstExt
	if ext == "." || ext == filepath.Ext(im.srcPath) {
		ext = ""
	}

	im.dstExt = ext
}

func (ia imgArgsNS) format(im *img) (args string) {
	var s []string

	add := func(k, v string) {
		s = append(s, fmt.Sprintf("%s=%s", k, v))
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

	if len(s) > 0 || im.dstExt != "" {
		// If there are no args, append a blank one so that there's always a
		// "." before the dstExt
		if len(s) == 0 {
			s = append(s, "")
		}

		ext := im.dstExt
		if ext == "" {
			ext = filepath.Ext(im.srcPath)
		}

		s = append(s, strings.Trim(ext, "."))
	}

	args = strings.Join(s, ".")

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
