package acryliclib

import (
	"sync/atomic"
	"time"
)

// BuildStats contains interesting information about a single build of the
// site.
type BuildStats struct {
	Duration time.Duration
	Pages    uint32
	JS       uint32
	CSS      uint32
	Imgs     uint32
	Blobs    uint32
}

// Build builds the site specified by cfg
func Build(cfg Config) (BuildStats, Errors) {
	cfg.setDefaults()

	startTime := time.Now()
	s, errs := newSite(&cfg).build()
	s.setRunTime(time.Now().Sub(startTime))

	return s, errs
}

func (bs *BuildStats) setRunTime(d time.Duration) {
	bs.Duration = d
}

func (bs *BuildStats) addPage() {
	atomic.AddUint32(&bs.Pages, 1)
}

func (bs *BuildStats) addJS() {
	atomic.AddUint32(&bs.JS, 1)
}

func (bs *BuildStats) addCSS() {
	atomic.AddUint32(&bs.CSS, 1)
}

func (bs *BuildStats) addImg() {
	atomic.AddUint32(&bs.Imgs, 1)
}

func (bs *BuildStats) addBlob() {
	atomic.AddUint32(&bs.Blobs, 1)
}
