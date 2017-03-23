package acrylic

//gocovr:skip-file

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/thatguystone/cog/stringc"
)

// A Bin is a go binary that needs to be built and run
type Bin struct {
	BuildCmd []string
	RunCmd   []string
	ProxyTo  string   // Address to proxy requests to
	AddlExts []string // Additional extensions to use to check for changes
	once     sync.Once
	proxy    Proxy
	changed  chan struct{}
	cmd      *exec.Cmd
	cmdErr   chan error

	rwmtx sync.RWMutex
	err   error
}

func (b *Bin) init() (first bool) {
	b.once.Do(func() {
		first = true

		b.proxy.To = b.ProxyTo
		b.cmdErr = make(chan error, 1)
		b.changed = make(chan struct{}, 2)

		b.rwmtx.Lock() // Lock, pending first build
		b.changed <- struct{}{}

		go b.run()
	})

	return
}

func (b *Bin) run() {
	first := true

	for {
		select {
		case <-b.changed:
			if !first {
				b.rwmtx.Lock()
			}
			first = false

			b.err = b.rebuild()
			b.rwmtx.Unlock()

			if b.err != nil {
				log.Printf("[bin] %s rebuild failed:\n%v",
					b.RunCmd[0],
					stringc.Indent(b.err.Error(), indent))
			}

		case err := <-b.cmdErr:
			b.rwmtx.Lock()
			b.err = err
			b.rwmtx.Unlock()

			log.Printf("[bin] %v", err)
		}
	}
}

func (b *Bin) rebuild() error {
	b.term()

	if len(b.BuildCmd) > 0 {
		out, err := exec.Command(b.BuildCmd[0], b.BuildCmd[1:]...).CombinedOutput()
		if err != nil {
			return errors.New(string(out))
		}
	}

	b.cmd = exec.Command(b.RunCmd[0], b.RunCmd[1:]...)
	b.cmd.Stdout = os.Stdout
	b.cmd.Stderr = os.Stderr
	go func() {
		err := b.cmd.Run()
		b.cmdErr <- fmt.Errorf("exited unexpectedly: %v", err)
	}()

	ready := b.proxy.pollReady(5 * time.Second)
	for {
		select {
		case err := <-b.cmdErr:
			return err

		case err := <-ready:
			if err != nil {
				b.term()
			}
			return err
		}
	}
}

func (b *Bin) term() {
	if b.cmd == nil {
		return
	}

	// Try to be nice
	b.cmd.Process.Signal(os.Interrupt)

	for {
		if b.cmd == nil || b.cmd.ProcessState != nil {
			b.cmd = nil
			return
		}

		select {
		case <-b.cmdErr:
		case <-time.After(100 * time.Millisecond):
			b.cmd.Process.Kill()
		}
	}
}

// Changed implements Watcher
func (b *Bin) Changed(evs WatchEvents) {
	changed := evs.HasExt(".go")
	for _, ext := range b.AddlExts {
		changed = changed || evs.HasExt(ext)
	}

	if !b.init() && changed {
		b.changed <- struct{}{}
	}
}

// ServeHTTP implements http.Handler
func (b *Bin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.init()

	b.rwmtx.RLock()
	defer b.rwmtx.RUnlock()

	b.proxy.Serve(b.err, w, r)
}
