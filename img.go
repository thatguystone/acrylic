package acrylic

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/stringc"
)

type img struct {
	srcPath      string // Path to src image
	resolvedName string // Name that should be used for this image

	// If this image is at its final path. If not, then a redirect is
	// necessary to get to the final path.
	isFinalPath bool

	w, h    int // If both == 0, just means original size
	oW, oH  int // Original value from args
	crop    bool
	quality int // 100 == original quality
	density int
	dstExt  string // Extension of dst (if not same as src)
}

const imgArgSep = "@"

// Parse up the image arguments
func newImg(src string, form url.Values) (*img, error) {
	im := &img{
		quality: 100,
		density: 1,
	}

	base := filepath.Base(src)
	if src == "" {
		base = ""
	}

	srcName, err := im.parseName(base)
	im.srcPath = filepath.Join(src, "..", srcName)
	if err != nil {
		return nil, err
	}

	// Allow QS args to override current args in path in order to make scaled
	// images composable
	usedForm, err := im.parseForm(form)
	if err != nil {
		return nil, err
	}

	imgArgs.postParse(im)

	args := imgArgs.format(im)
	if args == "" {
		im.resolvedName = srcName
	} else {
		im.resolvedName = fmt.Sprintf("%s%s%s",
			srcName,
			imgArgSep, args)
	}

	im.isFinalPath = !usedForm && im.resolvedName == base

	return im, nil
}

// Determine if a scale operation is necessary. It's only not necessary when
// no operations have been given.
func (im *img) needsScale() bool {
	return im.w != 0 || im.h != 0 ||
		im.crop ||
		im.quality != 100 ||
		im.density > 1 ||
		im.dstExt != ""
}

func (im *img) parseName(name string) (srcName string, err error) {
	sepI := strings.LastIndex(name, imgArgSep)
	if sepI == -1 {
		srcName = name
	} else {
		srcName = name[:sepI]
		err = imgArgs.parseNameArgs(im, name[sepI+1:])
	}

	return
}

func (im *img) parseForm(form url.Values) (bool, error) {
	return imgArgs.parseForm(im, form)
}

func (im *img) scale(dstPath string) error {
	err := cfs.CreateParents(dstPath)

	if err == nil {
		args := append(imgArgs.cmdArgs(im), im.srcPath, dstPath)

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
		err = fmt.Errorf("failed to scale %s: %v", dstPath, err)
	}

	return err
}
