package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"sync"
)

// Errors is a slice of all errors that occurred while processing
type aErrors []aError

// Error contains user errors that occurred while processing the given content
type aError struct {
	path string
	errs []error
}

type errs struct {
	mtx sync.Mutex
	s   aErrors
}

func (e *errs) has() bool {
	return len(e.s) > 0
}

func (e *errs) add(path string, err error) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	at := sort.Search(len(e.s), func(i int) bool {
		return e.s[i].path >= path
	})

	if at >= len(e.s) || e.s[at].path != path {
		e.s = append(e.s, aError{})
		copy(e.s[at+1:], e.s[at:])
		e.s[at] = aError{
			path: path,
		}
	}

	e.s[at].errs = append(e.s[at].errs, err)
}

func (e *errs) String() string {
	b := bytes.Buffer{}

	for _, e := range e.s {
		e.dump(&b)
	}

	return b.String()
}

func (e aError) String() string {
	b := bytes.Buffer{}
	e.dump(&b)
	return b.String()
}

func (e aError) dump(w io.Writer) bool {
	if len(e.errs) == 0 {
		return true
	}

	fmt.Fprintf(w, "from: %s\n", e.path)

	for _, err := range e.errs {
		fmt.Fprintf(w, "\t%v\n", err)
	}

	return false
}
