package imgscale

import (
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"strings"

	"github.com/goji/param"
	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/cog/stringc"
)

type args struct {
	W, H    *int // Dimensions
	D       int  // Density (for HiDPI devices)
	Q       *int // Image quality
	Crop    bool // How to crop the image
	Gravity cropGravity
	Ext     string // Convert to this format
}

func newArgs(u *url.URL) (args args, err error) {
	// Default to current extension
	args.Ext = path.Ext(u.Path)

	err = param.Parse(u.Query(), &args)
	if err != nil {
		return
	}

	if args.D != 0 {
		if args.W != nil {
			*args.W *= args.D
		}

		if args.H != nil {
			*args.H *= args.D
		}

		// Only used to simplify scaling externally
		args.D = 0
	}

	if args.W == nil && args.H == nil {
		args.Crop = false
	}

	if !args.Crop {
		args.Gravity = 0
	}

	if !strings.HasPrefix(args.Ext, ".") {
		args.Ext = "." + args.Ext
	}

	return
}

func (args *args) nameSuffix() string {
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

	buff.WriteString(args.Ext)

	return buff.String()
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

func (args *args) scale(srcPath, dstPath string) error {
	cmdArgs := []string{
		"convert",
		"-strip", // Scrub the image by default
		srcPath,
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

	cmdArgs = append(cmdArgs, dstPath)

	out, err := exec.Command("gm", cmdArgs...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("scale: %v\n%s",
			err.Error(),
			stringc.Indent(string(out), internal.Indent))
	}

	return err
}
