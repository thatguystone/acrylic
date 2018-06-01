package acrylic

import (
	"encoding"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goji/param"
	"github.com/thatguystone/acrylic/crawl"
	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/stringc"
)

type ImageScaler struct {
	Root  string // Root path to look for images
	Cache string // Where to cache scaled images
}

func (*ImageScaler) Start(*Watch)        {}
func (*ImageScaler) Changed(WatchEvents) {}

func (s *ImageScaler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var opts imgOpts

	err := opts.parse(r.URL.Query())
	if err != nil {
		HTTPError(w, err.Error(), http.StatusBadRequest)
		return
	}

	origPath := filepath.Join(s.Root, r.URL.Path)
	origInfo, err := os.Stat(origPath)
	if err != nil {
		status := http.StatusInternalServerError
		if os.IsNotExist(err) {
			status = http.StatusNotFound
		}

		HTTPError(w, err.Error(), status)
		return
	}

	cachePath := filepath.Join(s.Cache, r.URL.Path, opts.query())
	cacheInfo, err := os.Stat(cachePath)
	if err != nil && !os.IsNotExist(err) {
		HTTPError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache's ModTime is always synced with orig's, so if unchanged, then cache
	// is still good
	if cacheInfo == nil || !origInfo.ModTime().Equal(cacheInfo.ModTime()) {
		err := s.scale(opts, origPath, cachePath, origInfo.ModTime())
		if err != nil {
			HTTPError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	crawl.Variant(w, opts.variantName(r.URL.Path))
	crawl.ServeFile(w, r, cachePath)
}

func (s *ImageScaler) scale(
	opts imgOpts, src, dst string, origMod time.Time) (err error) {

	// Use a temp file to implement atomic cache writes; specifically, write to
	// the temp file first, then if everything is good, replace any existing
	// cache file with the temp file (on Linux, at least, this is atomic).
	tmpF, err := ioutil.TempFile(filepath.Dir(dst), "acrylic-")
	if err != nil {
		return err
	}

	tmpPath := tmpF.Name()
	tmpF.Close()

	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	err = opts.scale(src, tmpPath)
	if err != nil {
		return err
	}

	err = os.Chtimes(tmpPath, time.Now(), origMod)
	if err != nil {
		return err
	}

	err = os.Rename(tmpPath, dst)
	return
}

type imgOpts struct {
	W, H    *int // Dimensions
	D       int  // Density (for HiDPI devices)
	Q       *int // Image quality
	Crop    bool // How to crop the image
	Gravity cropGravity
	Ext     *string // Convert to this format
}

func (opts *imgOpts) parse(qs url.Values) error {
	err := param.Parse(qs, opts)
	if err != nil {
		return err
	}

	if opts.D != 0 {
		if opts.W != nil {
			*opts.W *= opts.D
		}

		if opts.H != nil {
			*opts.H *= opts.D
		}

		// Only used to simplify usage externally
		opts.D = 0
	}

	if opts.W == nil && opts.H == nil {
		opts.Crop = false
	}

	if !opts.Crop {
		opts.Gravity = 0
	}

	if opts.Ext != nil && !strings.HasPrefix(*opts.Ext, ".") {
		*opts.Ext = "." + *opts.Ext
	}

	return nil
}

func (opts *imgOpts) query() string {
	qs := make(url.Values)

	if opts.W != nil {
		qs.Set("w", strconv.Itoa(*opts.W))
	}

	if opts.H != nil {
		qs.Set("h", strconv.Itoa(*opts.H))
	}

	if opts.Q != nil {
		qs.Set("q", strconv.Itoa(*opts.Q))
	}

	if opts.Crop {
		qs.Set("crop", "1")

		if opts.Gravity != center {
			qs.Set("gravity", opts.Gravity.shortName())
		}
	}

	if opts.Ext != nil {
		qs.Set("ext", *opts.Ext)
	}

	if len(qs) == 0 {
		return ""
	}

	return "?" + qs.Encode()
}

func (opts *imgOpts) variantName(path string) string {
	var buff strings.Builder

	dims := opts.getDims()
	if dims != "" {
		buff.WriteString("-")
		buff.WriteString(dims)
	}

	if opts.Q != nil {
		fmt.Fprintf(&buff, "-q%d", *opts.Q)
	}

	if opts.Crop {
		buff.WriteString("-c")

		if opts.Gravity != center {
			buff.WriteString(opts.Gravity.shortName())
		}
	}

	if opts.Ext != nil {
		buff.WriteString(*opts.Ext)
	} else {
		buff.WriteString(filepath.Ext(path))
	}

	return cfs.ChangeExt(path, buff.String())
}

func (opts *imgOpts) getDims() string {
	switch {
	case opts.W == nil && opts.H == nil:
		return ""

	case opts.W == nil:
		return fmt.Sprintf("x%d", *opts.H)

	case opts.H == nil:
		return fmt.Sprintf("%dx", *opts.W)

	default:
		return fmt.Sprintf("%dx%d", *opts.W, *opts.H)
	}
}

func (opts *imgOpts) scale(src, dst string) error {
	args := []string{
		"convert",
		"-strip", // Scrub the image by default
		src,
	}

	dims := opts.getDims()
	if dims != "" {
		if opts.Crop {
			dims += "^"

			args = append(args,
				"-gravity", opts.Gravity.String(),
				"-extent", dims)
		}

		args = append(args, "-scale", dims)
	}

	if opts.Q != nil {
		args = append(args,
			"-quality", fmt.Sprintf("%d", *opts.Q))
	}

	err := cfs.CreateParents(dst)
	if err != nil {
		return err
	}

	args = append(args, dst)

	out, err := exec.Command("gm", args...).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("scale: %v\n%s",
			err.Error(),
			stringc.Indent(string(out), crawl.ErrIndent))
	}

	return err
}

type cropGravity int

const (
	center cropGravity = iota // Center is the default
	northWest
	north
	northEast
	west
	east
	southWest
	south
	southEast
)

var (
	_ encoding.TextMarshaler   = cropGravity(0)
	_ encoding.TextUnmarshaler = (*cropGravity)(nil)
	_ fmt.Stringer             = cropGravity(0)
)

func (g cropGravity) MarshalText() ([]byte, error) {
	return []byte(g.shortName()), nil
}

func (g *cropGravity) UnmarshalText(b []byte) error {
	switch strings.ToLower(string(b)) {
	case "c", "center":
		*g = center
	case "nw", "northwest":
		*g = northWest
	case "n", "north":
		*g = north
	case "ne", "northeast":
		*g = northEast
	case "w", "west":
		*g = west
	case "e", "east":
		*g = east
	case "sw", "southwest":
		*g = southWest
	case "s", "south":
		*g = south
	case "se", "southeast":
		*g = southEast
	default:
		return fmt.Errorf("unrecognized gravity: %q", string(b))
	}

	return nil
}

func (g cropGravity) shortName() string {
	switch g {
	case center:
		return "c"
	case northWest:
		return "nw"
	case north:
		return "n"
	case northEast:
		return "ne"
	case west:
		return "w"
	case east:
		return "e"
	case southWest:
		return "sw"
	case south:
		return "s"
	case southEast:
		return "se"
	default:
		panic(fmt.Errorf("unrecognized gravity: %d", g))
	}
}

func (g cropGravity) String() string {
	switch g {
	case center:
		return "Center"
	case northWest:
		return "NorthWest"
	case north:
		return "North"
	case northEast:
		return "NorthEast"
	case west:
		return "West"
	case east:
		return "East"
	case southWest:
		return "SouthWest"
	case south:
		return "South"
	case southEast:
		return "SouthEast"
	default:
		panic(fmt.Errorf("unrecognized gravity: %d", g))
	}
}
