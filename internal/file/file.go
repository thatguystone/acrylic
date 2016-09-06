package file

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thatguystone/acrylic/internal/afs"
	"github.com/thatguystone/acrylic/internal/state"
	"github.com/thatguystone/acrylic/internal/strs"
	"github.com/thatguystone/cog"
	"github.com/thatguystone/cog/cfs"
)

// F is a single file
type F struct {
	Info     os.FileInfo
	Src      string                 // File's source
	Dst      string                 // Destination path of generated file
	Cat      string                 // File's category
	Title    string                 // Human-readable title
	Name     string                 // Name of file
	Date     time.Time              // File's date
	SortName string                 // Name to use for sorting
	URL      string                 // Absolute URL to file
	Meta     map[string]interface{} // File metadata
}

func New(srcPath, srcDir string, isPage bool, st *state.S) (f F) {
	info, err := os.Stat(srcPath)
	cog.Must(err, "failed to stat %s", srcPath)
	return NewWithInfo(info, srcPath, srcDir, isPage, st)
}

func NewWithInfo(info os.FileInfo, srcPath, srcDir string, isPage bool, st *state.S) (f F) {
	f.Info = info
	f.Src = srcPath
	f.Name = afs.DropRoot(srcDir, f.Src)

	if strings.Count(f.Name, "/") == 0 {
		f.URL = "/" + filepath.Clean(f.Name)
		f.Dst = filepath.Join(st.Cfg.PublicDir, f.Name)

		f.SortName = f.Name
		f.Name = cfs.ChangeExt(f.Name, "")
		return
	}

	parts := strings.Split(f.Name, "/")
	if len(parts) == 2 {
		f.Cat = parts[0]
		f.SortName = parts[1]
		f.Date, f.Name = parseName(parts[1])
	} else {
		last := parts[len(parts)-2]
		_, ok := strs.ToDate(last)
		if ok {
			f.Date, f.Name = parseName(last)
			f.Cat = filepath.Join(parts[0 : len(parts)-2]...)
		} else {
			last = parts[len(parts)-1]
			f.Date, f.Name = parseName(last)
			f.Cat = filepath.Join(parts[0 : len(parts)-1]...)
		}

		f.SortName = last
	}

	nameExt := f.Name
	f.Name = cfs.ChangeExt(f.Name, "")
	if f.Date.IsZero() {
		f.URL = f.Cat
		if nameExt != "index.html" && isPage {
			f.URL = filepath.Join(f.URL, f.Name)
		}
	} else {
		f.URL = filepath.Join(f.Cat, f.Date.Format("2006/01/02"), f.Name)
	}

	dst := filepath.Join(st.Cfg.PublicDir, f.URL)
	if isPage {
		f.Dst = filepath.Join(dst, "index.html")
		f.URL += "/"
	} else {
		base := filepath.Base(f.Src)
		f.Dst = filepath.Join(dst, base)
		f.URL = filepath.Join(f.URL, base)
		if f.SortName != base {
			f.SortName = filepath.Join(f.SortName, base)
		}
	}

	f.URL = "/" + f.URL

	return
}

func parseName(name string) (time.Time, string) {
	t, ok := strs.ToDate(name)
	if !ok {
		return t, name
	}

	return t, strings.Trim(name[len(strs.DateFormat):], "-")
}
