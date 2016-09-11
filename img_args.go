package acrylic

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type imgArgsSlice []imgArg

// This could be an interface, but doing that makes the code REALLY verbose
type imgArg struct {
	name   string
	parse  func(im *img, val string) error
	format func(im *img) string
}

// All known arguments, in a normalized order that will produce consistent
// names
var imgArgs = imgArgsSlice{
	imgArg{
		name: "w",
		parse: func(im *img, val string) (err error) {
			im.oW, err = imgIntArg(val)
			return
		},
		format: func(im *img) string {
			if im.w == 0 {
				return ""
			}

			return strconv.Itoa(im.w)
		},
	},
	imgArg{
		name: "h",
		parse: func(im *img, val string) (err error) {
			im.oH, err = imgIntArg(val)
			return
		},
		format: func(im *img) string {
			if im.h == 0 {
				return ""
			}

			return strconv.Itoa(im.h)
		},
	},
	imgArg{
		name: "c",
		parse: func(im *img, val string) (err error) {
			im.crop = val != ""
			return
		},
		format: func(im *img) string {
			if !im.crop {
				return ""
			}

			return "t"
		},
	},
	imgArg{
		name: "q",
		parse: func(im *img, val string) (err error) {
			im.quality, err = imgIntArg(val)
			if err == nil && im.quality == 0 {
				im.quality = 100
			}
			return
		},
		format: func(im *img) string {
			if im.quality == 100 {
				return ""
			}

			return strconv.Itoa(im.quality)
		},
	},
	imgArg{
		name: "d",
		parse: func(im *img, val string) (err error) {
			im.density, err = imgIntArg(val)
			if err == nil && im.density == 0 {
				im.density = 1
			}

			return
		},
		format: func(im *img) string {
			// Density just causes W and H to be scaled
			return ""
		},
	},
	imgArg{
		name: "srcExt",
		parse: func(im *img, val string) (err error) {
			if !strings.HasPrefix(val, ".") {
				val = "." + val
			}

			im.srcExt = val

			return
		},
		format: func(im *img) string {
			if im.srcExt == im.dstExt {
				return ""
			}

			return im.srcExt
		},
	},
}

func (ih imgArgsSlice) parse(
	im *img,
	args url.Values) (parsedAny bool, err error) {

	for _, arg := range ih {
		val := args.Get(arg.name)
		if val == "" {
			continue
		}

		parsedAny = true

		err = arg.parse(im, val)
		if err != nil {
			err = errors.Wrapf(err, "invalid %s", arg.name)
			return
		}
	}

	return
}

func (ih imgArgsSlice) format(im *img) (args string) {
	for _, arg := range ih {
		val := arg.format(im)

		if val != "" {
			// QueryEscape(): though it's part of the path, it's parsed with
			// ParseQuery, so just keep things consistent.
			args += fmt.Sprintf("%s=%s;", arg.name, url.QueryEscape(val))
		}
	}

	args = strings.Trim(args, ";")

	return
}

func (ih imgArgsSlice) cmdArgs(im *img) (args []string) {
	dims := ""

	// Use something like '400x' to scale to a width of 400
	if im.w != 0 {
		dims += strconv.Itoa(im.w)
	}

	dims += "x"

	if im.h != 0 {
		dims += strconv.Itoa(im.h)
	}

	scaleDims := dims

	if im.crop {
		scaleDims += "^"

		args = append(args,
			"-gravity", "center",
			"-extent", dims)
	}

	if dims != "x" {
		args = append(args,
			"-scale", scaleDims)
	}

	if im.quality != 100 {
		args = append(args,
			"-quality", fmt.Sprintf("%d", im.quality))
	}

	return
}

func imgIntArg(val string) (int, error) {
	i, err := strconv.Atoi(val)
	if err == nil && i < 0 {
		err = fmt.Errorf("arg must be > 0")
	}

	return i, err
}
