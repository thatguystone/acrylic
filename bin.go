package acrylic

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

// A Bin builds a binary and proxies all traffic to it
type Bin struct {
	BuildCmd []string // Command to build
	RunCmd   []string // Command to run
	ProxyTo  string   // Address to proxy requests to
	Exts     []string // Extensions to use to check for changes

	changed chan struct{}

	rwmtx  sync.RWMutex
	cmd    *exec.Cmd
	err    error
	cmdErr chan error
	proxy  *Proxy
}

func (b *Bin) Start(w *Watch) {
	proxy, err := NewProxy(b.ProxyTo)
	if err != nil {
		panic(err)
	}

	b.proxy = proxy
	b.changed = make(chan struct{}, 1)
	b.changed <- struct{}{}

	// Lock, pending first build
	b.rwmtx.Lock()

	go b.run()
}

func (b *Bin) Changed(evs WatchEvents) {
	if b.shouldRebuild(evs) {
		b.changed <- struct{}{}
	}
}

func (b *Bin) shouldRebuild(evs WatchEvents) bool {
	for _, ext := range b.Exts {
		if evs.HasExt(ext) {
			return true
		}
	}

	return false
}

func (b *Bin) run() {
	first := true

	for {
		select {
		case <-b.changed:
			// Term before locking: long-running requests can block this otherwise
			b.term()

			if !first {
				b.rwmtx.Lock()
			}
			first = false

			start := time.Now()
			log.Printf("I: bin %s: rebuilding...\n", b.RunCmd[0])
			b.err = b.rebuild()

			b.rwmtx.Unlock()

			if b.err == nil {
				log.Printf("I: bin %s: rebuild took %s\n",
					b.RunCmd[0], time.Since(start))
			} else {
				log.Printf("E: bin %s: rebuild failed:\n%v",
					b.RunCmd[0], stringc.Indent(b.err.Error(), indent))
			}

		case err := <-b.cmdErr:
			b.rwmtx.Lock()
			b.err = err
			b.rwmtx.Unlock()

			log.Printf("E: bin %s: exited:\n%v",
				b.RunCmd[0], stringc.Indent(b.err.Error(), indent))
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
		select {
		case <-b.cmdErr:
			if b.cmd.ProcessState != nil {
				b.cmd = nil
				return
			}

		case <-time.After(100 * time.Millisecond):
			b.cmd.Process.Kill()
		}
	}
}

func (b *Bin) rebuild() error {
	if len(b.BuildCmd) > 0 {
		cmd := exec.Command(b.BuildCmd[0], b.BuildCmd[1:]...)
		out, err := cmd.CombinedOutput()
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

	ready := b.proxy.PollReady(5 * time.Second)
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

func (b *Bin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.rwmtx.RLock()
	defer b.rwmtx.RUnlock()

	if b.err != nil {
		sendError(w, b.err)
	} else {
		b.proxy.ServeHTTP(w, r)
	}
}
