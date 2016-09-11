package acrylic

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/stringc"
)

type img struct {
	src     string // Name given in newImg()
	srcBase string // Name without args, params, or ext

	// If this image is at its final path. If not, then a redirect is
	// necessary to get to the final path.
	isFinalPath bool

	w, h    int // If both == 0, just means original size
	oW, oH  int // Original value from args
	crop    bool
	quality int // 100 == original quality
	density int
	srcExt  string // Extension of source image
	dstExt  string // Extension of converted image
}

var reImgNameArgs = regexp.MustCompile(`^([^:]*):?(.*?)(\.\w+)?$`)

// Parse up the image arguments
func newImg(src string, form url.Values) (im *img, err error) {
	im = &img{
		src:     src,
		quality: 100,
		density: 1,
		srcExt:  filepath.Ext(src),
		dstExt:  filepath.Ext(src),
	}

	im.isFinalPath, err = im.parseForm(form)
	if err == nil {
		err = im.parseName(filepath.Base(src))
	}

	if err != nil {
		im = nil
	}

	return
}

// Determine if a scale operation is necessary. It's only not necessary when
// no operations have been given.
func (im *img) needsScale() bool {
	return im.w != 0 || im.h != 0 ||
		im.crop ||
		im.quality != 100 ||
		im.density > 1
}

func (im *img) srcPath() string {
	return filepath.Join(
		im.src, "..",
		fmt.Sprintf("%s%s", im.srcBase, im.srcExt))
}

func (im *img) scaledName() string {
	args := imgArgs.format(im)
	colon := ":"
	if len(args) == 0 {
		colon = ""
	}

	return fmt.Sprintf("%s%s%s%s",
		im.srcBase,
		colon, args,
		im.dstExt)
}

func (im *img) parseName(name string) (err error) {
	s := []string{"", cfs.DropExt(name), filepath.Ext(name)}
	if strings.Contains(name, ":") {
		s = reImgNameArgs.FindStringSubmatch(name)
	}

	cog.Assert(len(s) > 0, "invalid path? should never happen")

	im.srcBase = s[1]

	form, err := url.ParseQuery(s[2])
	if err == nil {
		_, err = im.parseForm(form)
	}

	return
}

func (im *img) parseForm(form url.Values) (parsedNone bool, err error) {
	parsedAny, err := imgArgs.parse(im, form)
	parsedNone = !parsedAny

	im.w = im.oW * im.density
	im.h = im.oH * im.density

	return
}

func (im *img) scale(dstPath string) error {
	err := cfs.CreateParents(dstPath)

	if err == nil {
		args := append(imgArgs.cmdArgs(im), im.srcPath(), dstPath)

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
		err = fmt.Errorf("failed to scale %s: %v", im.src, err)
	}

	return err
}
