package imgscale

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goji/param"
	"github.com/thatguystone/acrylic/crawl"
	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/stringc"
)

type args struct {
	W, H    *int // Dimensions
	D       int  // Density (for HiDPI devices)
	Q       *int // Image quality
	Crop    bool // How to crop the image
	Gravity cropGravity
	Ext     *string // Convert to this format
}

func (args *args) parse(qs url.Values) error {
	err := param.Parse(qs, args)
	if err != nil {
		return err
	}

	if args.D != 0 {
		if args.W != nil {
			*args.W *= args.D
		}

		if args.H != nil {
			*args.H *= args.D
		}

		// Only used to simplify usage externally
		args.D = 0
	}

	if args.W == nil && args.H == nil {
		args.Crop = false
	}

	if !args.Crop {
		args.Gravity = 0
	}

	if args.Ext != nil && !strings.HasPrefix(*args.Ext, ".") {
		*args.Ext = "." + *args.Ext
	}

	return nil
}

func (args *args) query() string {
	qs := make(url.Values)

	if args.W != nil {
		qs.Set("W", strconv.Itoa(*args.W))
	}

	if args.H != nil {
		qs.Set("H", strconv.Itoa(*args.H))
	}

	if args.Q != nil {
		qs.Set("Q", strconv.Itoa(*args.Q))
	}

	if args.Crop {
		qs.Set("Crop", "1")

		if args.Gravity != center {
			qs.Set("Gravity", args.Gravity.shortName())
		}
	}

	if args.Ext != nil {
		qs.Set("Ext", *args.Ext)
	}

	if len(qs) == 0 {
		return ""
	}

	return "?" + qs.Encode()
}

func (args *args) variantName(path string) string {
	var buff strings.Builder

	dims := args.getDims()
	if dims != "" {
		buff.WriteString("-")
		buff.WriteString(dims)
	}

	if args.Q != nil {
		fmt.Fprintf(&buff, "-q%d", *args.Q)
	}

	if args.Crop {
		buff.WriteString("-c")

		if args.Gravity != center {
			buff.WriteString(args.Gravity.shortName())
		}
	}

	buff.WriteString(args.getExt(path))

	return cfs.DropExt(path) + buff.String()
}

func (args *args) getDims() string {
	switch {
	case args.W == nil && args.H == nil:
		return ""

	case args.W == nil:
		return fmt.Sprintf("x%d", *args.H)

	case args.H == nil:
		return fmt.Sprintf("%dx", *args.W)

	default:
		return fmt.Sprintf("%dx%d", *args.W, *args.H)
	}
}

func (args *args) getExt(path string) string {
	if args.Ext != nil {
		return *args.Ext
	}

	return filepath.Ext(path)
}

func (args *args) getTempFilePattern(path string) string {
	return "acrylic-*" + args.getExt(path)
}

func (args *args) scale(src, dst string) error {
	cmdArgs := []string{
		"convert",
		"-strip", // Scrub the image by default
		src,
	}

	dims := args.getDims()
	if dims != "" {
		if args.Crop {
			dims += "^"

			cmdArgs = append(cmdArgs,
				"-gravity", args.Gravity.String(),
				"-extent", dims)
		}

		cmdArgs = append(cmdArgs, "-scale", dims)
	}

	if args.Q != nil {
		cmdArgs = append(cmdArgs,
			"-quality", fmt.Sprintf("%d", *args.Q))
	}

	cmdArgs = append(cmdArgs, dst)

	out, err := exec.Command("gm", cmdArgs...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("scale: %v\n%s",
			err.Error(),
			stringc.Indent(string(out), crawl.ErrIndent))
	}

	return err
}
