package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/fsnotify.v1"
)

type fnotify struct {
	watcher *fsnotify.Watcher
	rebuild chan struct{}
	dir     string
}

const rebuildDelay = 40 * time.Millisecond

func fwatch() *fnotify {
	fnot := &fnotify{
		rebuild: make(chan struct{}),
	}

	fnot.refreshWatcher()

	return fnot
}

func (fnot *fnotify) refreshWatcher() {
	if fnot.watcher != nil {
		fnot.watcher.Close()
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	fnot.watcher = w
}

func (fnot *fnotify) setDir(dir string) {
	if dir == "" || dir != fnot.dir {
		fnot.refreshWatcher()
		if dir == "" {
			return
		}
	}

	fnot.doRecursive(fnot.watcher, dir, true)
	go fnot.run(fnot.watcher)
}

func (fnot *fnotify) doRecursive(w *fsnotify.Watcher, p string, add bool) {
	mod := func(p string) {
		var err error
		if add {
			err = w.Add(p)
		} else {
			err = w.Remove(p)
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	var walk func(string)
	walk = func(p string) {
		f, err := os.Open(p)
		if err != nil {
			log.Fatal(err)
		}

		infos, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			log.Fatal(err)
		}

		for _, info := range infos {
			p := filepath.Join(p, info.Name())

			if info.IsDir() {
				mod(p)
				walk(p)
			}
		}
	}

	mod(p)
	walk(p)
}

func (fnot *fnotify) run(w *fsnotify.Watcher) {
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			select {
			case <-fnot.watcher.Errors:
			case fnot.rebuild <- struct{}{}:
			}

		case ev := <-fnot.watcher.Events:
			timer.Reset(rebuildDelay)

			info, err := os.Stat(ev.Name)
			if err != nil || !info.IsDir() {
				continue
			}

			switch ev.Op {
			case fsnotify.Create:
				fnot.doRecursive(w, ev.Name, true)

			case fsnotify.Remove:
				fnot.doRecursive(w, ev.Name, false)
			}

		case err := <-fnot.watcher.Errors:
			if err != nil {
				log.Fatal(err)
			}
			return
		}
	}
}
