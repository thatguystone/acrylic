package toner

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
)

// Error contains user errors that occurred while processing the given content
type Error struct {
	Path string
	Errs []error
}

type errs struct {
	mtx sync.Mutex
	s   []Error
}

func (e *errs) has() bool {
	return len(e.s) > 0
}

func (e *errs) add(path string, err error) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	at := sort.Search(len(e.s), func(i int) bool {
		return e.s[i].Path >= path
	})

	if at >= len(e.s) || e.s[at].Path != path {
		e.s = append(e.s, Error{})
		copy(e.s[at+1:], e.s[at:])
		e.s[at] = Error{
			Path: path,
		}
	}

	e.s[at].Errs = append(e.s[at].Errs, err)
}

func (e Error) String() string {
	if len(e.Errs) == 0 {
		return ""
	}

	b := bytes.Buffer{}

	fmt.Fprintf(&b, "from: %s\n", e.Path)

	for _, err := range e.Errs {
		fmt.Fprintf(&b, "\t%v\n", err)
	}

	return b.String()
}
