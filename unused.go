package main

import (
	"os"
	"sort"
)

type unused struct {
	files map[string]struct{}
}

func newUnused() *unused {
	return &unused{
		files: map[string]struct{}{},
	}
}

func (u *unused) add(path string) {
	u.files[path] = struct{}{}
}

func (u *unused) used(path string) {
	delete(u.files, path)
}

func (u *unused) remove() {
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
