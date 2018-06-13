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

	"github.com/thatguystone/acrylic/crawl"
	"github.com/thatguystone/cog/stringc"
)

// A Bin builds a binary and proxies all traffic to it
type BinConfig struct {
	BuildCmd []string // Command to build
	RunCmd   []string // Command to run
	ProxyTo  string   // Address to proxy requests to
	Exts     []string // Extensions to use to check for changes
}

type bin struct {
	cfg     BinConfig
	changed chan struct{}

	rwmtx  sync.RWMutex
	cmd    *exec.Cmd
	err    error
	cmdErr chan error
	proxy  *Proxy
}

func NewBin(cfg BinConfig) HandlerWatcher {
	proxy, err := NewProxy(cfg.ProxyTo)
	if err != nil {
		panic(err)
	}

	b := bin{
		cfg:     cfg,
		changed: make(chan struct{}, 1),
		proxy:   proxy,
	}

	// Lock, pending first build
	b.rwmtx.Lock()
	go b.run()

	b.changed <- struct{}{}

	return &b
}

func (b *bin) Changed(evs WatchEvents) {
	if b.shouldRebuild(evs) {
		b.changed <- struct{}{}
	}
}

func (b *bin) shouldRebuild(evs WatchEvents) bool {
	for _, ext := range b.cfg.Exts {
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
			// Term before locking: long-running requests can block this otherwise
			b.term()

			if !first {
				b.rwmtx.Lock()
			}
			first = false

			start := time.Now()
			log.Printf("I: bin %s: rebuilding...\n", b.cfg.RunCmd[0])
			b.err = b.rebuild()

			b.rwmtx.Unlock()

			if b.err == nil {
				log.Printf("I: bin %s: rebuild took %s\n",
					b.cfg.RunCmd[0], time.Since(start))
			} else {
				log.Printf("E: bin %s: rebuild failed:\n%v",
					b.cfg.RunCmd[0], stringc.Indent(b.err.Error(), crawl.ErrIndent))
			}

		case err := <-b.cmdErr:
			b.rwmtx.Lock()
			b.err = err
			b.rwmtx.Unlock()

			log.Printf("E: bin %s: exited:\n%v",
				b.cfg.RunCmd[0], stringc.Indent(b.err.Error(), crawl.ErrIndent))
		}
	}
}

func (b *bin) term() {
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

func (b *bin) rebuild() error {
	if len(b.cfg.BuildCmd) > 0 {
		cmd := exec.Command(b.cfg.BuildCmd[0], b.cfg.BuildCmd[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return errors.New(string(out))
		}
	}

	b.cmd = exec.Command(b.cfg.RunCmd[0], b.cfg.RunCmd[1:]...)
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

func (b *bin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.rwmtx.RLock()
	defer b.rwmtx.RUnlock()

	if b.err != nil {
		HTTPError(w, b.err.Error(), http.StatusInternalServerError)
	} else {
		b.proxy.ServeHTTP(w, r)
	}
}
