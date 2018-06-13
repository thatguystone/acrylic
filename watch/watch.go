// Package watch implements a filesystem watcher
package watch

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rjeczalik/notify"
)

// A Watcher receives notifications of changes
type Watcher interface {
	Changed(evs Events)
}

// Watch wraps file system watchers and provides a nice interface to receive
// change notifications
type Watch struct {
	evs      chan notify.EventInfo
	watchers chan Watcher
}

// New creates a new Watch that monitors the given paths
func New(paths ...string) *Watch {
	w := &Watch{
		evs:      make(chan notify.EventInfo, 16),
		watchers: make(chan Watcher, 1),
	}

	go w.run()
	w.Watch(paths...)
	return w
}

// Watch adds additional paths to the watch
func (w *Watch) Watch(paths ...string) {
	for _, path := range paths {
		err := notify.Watch(path, w.evs, notify.All)
		if err != nil {
			panic(fmt.Errorf("failed to watch %q: %v", path, err))
		}
	}
}

// Notify notifies the given Watcher of changes as they happen
func (w *Watch) Notify(wr Watcher) {
	if wr != nil {
		w.watchers <- wr
	}
}

// Stop terminates this instance
func (w *Watch) Stop() {
	notify.Stop(w.evs)
	close(w.evs)
}

func (w *Watch) run() {
	delay := time.NewTimer(time.Hour)
	delay.Stop()

	var evs Events
	var watchers []Watcher

	for {
		select {
		case wr := <-w.watchers:
			watchers = append(watchers, wr)

		case ev := <-w.evs:
			if ev == nil {
				return
			}

			evs = append(evs, ev)
			delay.Reset(25 * time.Millisecond)

		case <-delay.C:
			for _, wr := range watchers {
				wr.Changed(evs)
			}

			evs = nil
		}
	}
}

// Events is a collection of change events
type Events []notify.EventInfo

// HasExt checks if any event path has the given extension
func (evs Events) HasExt(ext string) bool {
	for _, ev := range evs {
		if filepath.Ext(ev.Path()) == ext {
			return true
		}
	}

	return false
}
