package acrylic

import (
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
	"github.com/thatguystone/cog"
)

// Watches collects a bunch of directory watches into 1
type Watches struct {
	ch       chan notify.EventInfo
	watchers chan Watcher
}

// A Watcher receives notifications of changes
type Watcher interface {
	Changed(evs WatchEvents)
}

// Watch watches the given path
func Watch(path string) Watches {
	w := Watches{
		ch:       make(chan notify.EventInfo, 16),
		watchers: make(chan Watcher, 4),
	}
	w.Watch(path)
	go w.run()
	return w
}

// Watch adds a path to the watches
func (w *Watches) Watch(path string) {
	err := notify.Watch(path, w.ch, notify.All)
	cog.Must(err, "failed to watch for changes")
}

// Stop stops the watcher and cleans up
func (w *Watches) Stop() {
	close(w.watchers)
}

// Notify adds a Watcher to those notified on change.
func (w *Watches) Notify(r Watcher) {
	if r != nil {
		w.watchers <- r
	}
}

func (w *Watches) run() {
	delay := time.NewTimer(time.Hour)
	delay.Stop()

	var evs WatchEvents
	var watchers []Watcher

	for {
		select {
		case r := <-w.watchers:
			if r == nil {
				return
			}

			watchers = append(watchers, r)

		case ev := <-w.ch:
			evs = append(evs, ev)
			delay.Reset(25 * time.Millisecond)

		case <-delay.C:
			for _, r := range watchers {
				r.Changed(evs)
			}

			evs = nil
		}
	}
}

// WatchEvents is a collection of change events
type WatchEvents []notify.EventInfo

// HasExt checks if any event path has the given extension
func (evs WatchEvents) HasExt(ext string) bool {
	for _, ev := range evs {
		if filepath.Ext(ev.Path()) == ext {
			return true
		}
	}

	return false
}
