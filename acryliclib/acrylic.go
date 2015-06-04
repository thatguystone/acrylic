package acryliclib

import (
	"sync/atomic"
	"time"
)

// Acrylic represents a single site
type Acrylic struct {
	cfg Config
}

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

func New(cfg Config) *Acrylic {
	cfg.setDefaults()

	return &Acrylic{
		cfg: cfg,
	}
}

// Build builds the current site
func (t *Acrylic) Build() (BuildStats, Errors) {
	startTime := time.Now()
	s, errs := newSite(&t.cfg).build()
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
