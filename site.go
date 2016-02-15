package main

import "io"

type site struct {
	args    []string
	cfg     *config
	logOut  io.Writer
	baseDir string
	errs    errs
}

func (s *site) build() bool {
	ss := newSiteState(s)
	return ss.build()
}
