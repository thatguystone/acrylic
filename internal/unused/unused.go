package unused

import (
	"os"
	"sort"
	"sync"
)

// Unused tracks unused files
type U struct {
	mtx   sync.Mutex
	files map[string]struct{}
}

// New creates a new unused file track
func New() *U {
	return &U{
		files: map[string]struct{}{},
	}
}

// Add a potentially-unused file
func (u *U) Add(path string) {
	u.mtx.Lock()
	u.files[path] = struct{}{}
	u.mtx.Unlock()
}

// Used marks a file as Used
func (u *U) Used(path string) {
	u.mtx.Lock()
	delete(u.files, path)
	u.mtx.Unlock()
}

// Remove removes any files not marked as Used()
func (u *U) Remove() {
	paths := []string{}
	for path := range u.files {
		paths = append(paths, path)
	}

	// Sorted in reverse, this should make sure that any empty directories are
	// removed recursively
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths {
		os.Remove(path)
	}
}
