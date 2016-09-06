package fnotify

import (
	"os"
	"path/filepath"

	"github.com/thatguystone/cog"

	"gopkg.in/fsnotify.v1"
)

type N struct {
	C       chan string
	watcher *fsnotify.Watcher
	exitCh  chan struct{}
	dir     string
}

func New() *N {
	n := &N{
		C:      make(chan string, 8),
		exitCh: make(chan struct{}),
	}

	w, err := fsnotify.NewWatcher()
	cog.Must(err, "failed to create new watcher")

	n.watcher = w
	go n.run()

	return n
}

func (n *N) Exit() {
	close(n.exitCh)
	n.watcher.Close()
}

func (n *N) AddDir(dir string) {
	n.doRecursive(dir, true)
}

func (n *N) doRecursive(p string, add bool) {
	mod := func(p string) {
		if add {
			err := n.watcher.Add(p)
			cog.Must(err, "failed to add %s", p)
		} else {
			n.watcher.Remove(p)
		}
	}

	mod(p)
	filepath.Walk(p,
		func(path string, info os.FileInfo, err error) error {
			mod(path)
			return nil
		})
}

func (n *N) run() {
	for {
		select {
		case ev := <-n.watcher.Events:
			select {
			case <-n.exitCh:
				continue
			case n.C <- ev.Name:
			}

			switch ev.Op {
			case fsnotify.Create:
				n.doRecursive(ev.Name, true)

			case fsnotify.Remove:
				n.doRecursive(ev.Name, false)
			}

		case err := <-n.watcher.Errors:
			cog.Must(err, "encountered watcher error")
			return
		}
	}
}
