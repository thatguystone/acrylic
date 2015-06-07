package acrylib

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"sync"
)

// Errors is a slice of all errors that occurred while processing
type Errors []Error

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

func (es Errors) String() string {
	if len(es) == 0 {
		return ""
	}

	b := bytes.Buffer{}

	for _, e := range es {
		e.dump(&b)
	}

	return b.String()
}

func (e Error) String() string {
	b := bytes.Buffer{}
	e.dump(&b)
	return b.String()
}

func (e Error) dump(w io.Writer) {
	if len(e.Errs) == 0 {
		return
	}

	fmt.Fprintf(w, "from: %s\n", e.Path)

	for _, err := range e.Errs {
		fmt.Fprintf(w, "\t%v\n", err)
	}
}
