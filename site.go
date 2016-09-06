package main

import "github.com/thatguystone/cog/clog"

type site struct {
	args []string
	cfg  *config
	log  *clog.Logger
}

func (s *site) build() bool {
NOTIFY OF ANY UNUSED CONTENT + ASSETS ONCE BUILD IS COMPLETE?

	ss := newSiteState(s)
	return ss.build()
}
