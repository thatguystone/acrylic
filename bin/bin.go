// Package bin implements a binary runner, compiler, reloader, and proxy
package bin

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

	"github.com/thatguystone/acrylic/internal"
	"github.com/thatguystone/acrylic/proxy"
	"github.com/thatguystone/acrylic/watch"
	"github.com/thatguystone/cog/stringc"
)

type bin struct {
	buildCmd []string
	runCmd   []string
	exts     []string
	logf     func(string, ...interface{})
	changed  chan struct{}

	proc   *os.Process
	err    error
	cmdErr chan error
	proxy  *proxy.Proxy
	reqMtx sync.RWMutex
}

// New creates a new binary runner/builder/proxy
func New(proxyTarget string, runCmd []string, opts ...Option) http.Handler {
	b := &bin{
		runCmd:  runCmd,
		logf:    log.Printf,
		changed: make(chan struct{}, 1),
		cmdErr:  make(chan error),
	}

	for _, opt := range opts {
		opt.applyTo(b)
	}

	// Lock before listening to prevent anyone from getting a proxy error before
	// the first build has even started
	b.reqMtx.Lock()

	proxy, err := proxy.New(proxyTarget,
		proxy.ErrorLog(func(s ...interface{}) {
			b.logf("%s", fmt.Sprint(s...))
		}))
	if err != nil {
		panic(err)
	}

	b.proxy = proxy
	go b.run()

	b.changed <- struct{}{}

	return b
}

func (b *bin) Changed(evs watch.Events) {
	if b.shouldRebuild(evs) {
		b.changed <- struct{}{}
	}
}

func (b *bin) shouldRebuild(evs watch.Events) bool {
	for _, ext := range b.exts {
		if evs.HasExt(ext) {
			return true
		}
	}

	return false
}

func (b *bin) run() {
	first := true

	for {
		select {
		case <-b.changed:
			// Term before locking: long-running requests can block rebuilding
			// otherwise
			b.term()

			if !first {
				b.reqMtx.Lock()
			}
			first = false

			start := time.Now()
			b.logf("I: bin %s: rebuilding...\n", b.runCmd[0])
			b.err = b.rebuild()

			b.reqMtx.Unlock()

			if b.err == nil {
				b.logf("I: bin %s: rebuild took %s\n",
					b.runCmd[0], time.Since(start))
			} else {
				b.logf("E: bin %s: rebuild failed:\n%v",
					b.runCmd[0], stringc.Indent(b.err.Error(), internal.Indent))
			}

		case err := <-b.cmdErr:
			b.proc = nil

			b.reqMtx.Lock()
			b.err = err
			b.reqMtx.Unlock()

			b.logf("E: bin %s: exited:\n%v",
				b.runCmd[0], stringc.Indent(b.err.Error(), internal.Indent))
		}
	}
}

func (b *bin) term() {
	if b.proc == nil {
		return
	}

	// Try to be nice
	b.proc.Signal(os.Interrupt)

	for {
		select {
		case <-b.cmdErr:
			b.proc = nil
			return

		case <-time.After(100 * time.Millisecond):
			b.proc.Kill()
		}
	}
}

func (b *bin) rebuild() error {
	if len(b.buildCmd) > 0 {
		cmd := exec.Command(b.buildCmd[0], b.buildCmd[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return errors.New(string(out))
		}
	}

	cmd := exec.Command(b.runCmd[0], b.runCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		err := cmd.Wait()
		b.cmdErr <- fmt.Errorf("exited unexpectedly: %v", err)
	}()

	select {
	case err := <-b.proxy.PollReady(5 * time.Second):
		if err == nil {
			b.proc = cmd.Process
		} else {
			b.term()
		}
		return err

	case err := <-b.cmdErr:
		return err
	}
}

func (b *bin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.reqMtx.RLock()
	defer b.reqMtx.RUnlock()

	if b.err != nil {
		internal.HTTPError(w, b.err.Error(), http.StatusInternalServerError)
	} else {
		b.proxy.ServeHTTP(w, r)
	}
}
