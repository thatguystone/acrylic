package acrylic

import "net/http"

// staticDirs builds on top of http.Dir, except it searches in each dir given
// before giving up.
type staticDirs []string

func (ds staticDirs) Open(name string) (f http.File, err error) {
	for _, d := range ds {
		f, err = http.Dir(d).Open(name)
		if err == nil {
			return
		}
	}

	return
}
