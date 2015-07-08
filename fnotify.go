package main

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/fsnotify.v1"
)

type fnotify struct {
	watcher *fsnotify.Watcher
	exitCh  chan struct{}
	changed chan string
	dir     string
}

const changedDelay = 40 * time.Millisecond

func fwatch() *fnotify {
	fnot := &fnotify{
		exitCh:  make(chan struct{}),
		changed: make(chan string, 8),
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	fnot.watcher = w
	go fnot.run()

	return fnot
}

func (fnot *fnotify) close() {
	close(fnot.exitCh)
	fnot.watcher.Close()
}

func (fnot *fnotify) addDir(dir string) {
	fnot.doRecursive(dir, true)
}

func (fnot *fnotify) addFile(path string) {
	err := fnot.watcher.Add(path)
	if err != nil {
		panic(err)
	}
}

func (fnot *fnotify) doRecursive(p string, add bool) {
	mod := func(p string) {
		var err error
		if add {
			err = fnot.watcher.Add(p)
		} else {
			err = fnot.watcher.Remove(p)
		}

		if err != nil {
			panic(err)
		}
	}

	var walk func(string)
	walk = func(p string) {
		f, err := os.Open(p)
		if err != nil {
			panic(err)
		}

		infos, err := f.Readdir(-1)
		f.Close()
		if err != nil {
			panic(err)
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

func (fnot *fnotify) run() {
	for {
		select {
		case ev := <-fnot.watcher.Events:
			select {
			case <-fnot.exitCh:
				continue
			case fnot.changed <- ev.Name:
			}

			info, err := os.Stat(ev.Name)
			if err != nil || !info.IsDir() {
				continue
			}

			switch ev.Op {
			case fsnotify.Create:
				fnot.doRecursive(ev.Name, true)

			case fsnotify.Remove:
				fnot.doRecursive(ev.Name, false)
			}

		case err := <-fnot.watcher.Errors:
			if err != nil {
				panic(err)
			}
			return
		}
	}
}
