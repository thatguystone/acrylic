package afs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/thatguystone/acrylic/internal/state"
	"github.com/thatguystone/cog/cfs"
)

// Copy copies the src file to dst, only if they differ
func Copy(src, dst string) error {
	if !SrcChanged(src, dst) {
		return nil
	}

	return cfs.Copy(src, dst)
}

func CopyState(st *state.S, src, dst string) {
	err := Copy(src, dst)
	if err != nil {
		st.Errs.Errorf(src, "failed to copy to `%s`: %v", dst, err)
		return
	}

	info, err := os.Stat(src)
	if err != nil {
		st.Errs.Errorf(src, "failed to stat: %v", err)
		return
	}

	os.Chtimes(dst, info.ModTime(), info.ModTime())
	st.Unused.Used(dst)
}

// DropFirst removes the first directory from the path
func DropFirst(path string) string {
	if len(path) == 0 {
		return ""
	}

	idx := strings.IndexRune(path[1:], os.PathSeparator)
	if idx == -1 {
		return ""
	}

	return path[idx+2:]
}

// SrcChanged checks if the src and dst differ
func SrcChanged(src, dst string) bool {
	sstat, serr := os.Stat(src)
	dstat, derr := os.Stat(dst)
	if serr != nil || derr != nil {
		return true
	}

	return !dstat.ModTime().Equal(sstat.ModTime())
}

// DropRoot removes the root prefix from given path
func DropRoot(root, path string) string {
	root = filepath.Clean(root)
	root += "/"

	path = filepath.Clean(path)

	if strings.HasPrefix(path, root) {
		return path[len(root):]
	}

	return path
}
