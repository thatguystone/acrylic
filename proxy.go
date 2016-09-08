package acrylic

import (
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rjeczalik/notify"
	"github.com/thatguystone/cog/cfs"
	"github.com/thatguystone/cog/stringc"
)

// ProxyArgs collects the arguments to pass to Proxy().
//
// Since the proxy is meant for debugging, everything runs in debug mode.
type ProxyArgs struct {
	WatchDirs   []string // Dirs to watch recursively for changes
	AppPath     string   // What to build and run on change
	AppArgs     []string // Arguments to pass to App
	ProxyAddr   string   // Address to run the proxy on
	WebpackAddr string   // Address to run webpack on (defaut: ":9600")
	AppURL      string   // Full URL to reach the app at
}

type proxy struct {
	ProxyArgs

	evCh  chan notify.EventInfo
	close chan struct{}

	rwmtx sync.RWMutex
	err   error

	app struct {
		url *url.URL
		cmd *cmd
	}

	webpack struct {
		host, port string
	}
}

func newProxy(args ProxyArgs) (*proxy, error) {
	p := &proxy{
		ProxyArgs: args,

		// Make the channel buffered to ensure no event is dropped. Notify
		// will drop an event if the receiver is not able to keep up.
		evCh:  make(chan notify.EventInfo, 64),
		close: make(chan struct{}),
	}

	err := p.checkApp()
	if err == nil {
		err = p.checkWebpack()
	}

	if err != nil {
		p = nil
	}

	return p, err
}

// Run runs the proxy until the program is terminated
func (p *proxy) run() error {
	defer notify.Stop(p.evCh)

	for _, dir := range p.WatchDirs {
		dir = filepath.Join(dir, "...")

		// Modifications are platform-specific. If you're getting build
		// errors, create an fswatch_{platform}.go file and define your flags.
		err := notify.Watch(dir, p.evCh, notify.All|notifyModEv)
		if err != nil {
			return errors.Wrapf(err, "failed to watch %s", dir)
		}
	}

	defer close(p.close)

	go p.runWatcher()
	go p.runWebpack()

	return p.serve()
}

func (p *proxy) checkApp() error {
	url, err := url.Parse(p.AppURL)
	if err != nil {
		return errors.Wrap(err, "failed to parse AppURL")
	}

	_, _, err = net.SplitHostPort(url.Host)
	if err != nil {
		return fmt.Errorf("`AppURL` is missing port")
	}

	p.app.url = url

	base := filepath.Base(p.AppPath)
	p.app.cmd = command("./"+cfs.DropExt(base), p.AppArgs...)

	return nil
}

func (p *proxy) checkWebpack() error {
	addr := p.WebpackAddr
	if addr == "" {
		addr = ":9600"
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.Wrap(err, "WebpackAddr")
	}

	p.webpack.host = host
	p.webpack.port = port

	setWebpackEnv(host, port)

	return nil
}

func (p *proxy) runWatcher() {
	// Be sure app is dead when exiting
	defer p.app.cmd.term()

	// Fire off build immediately
	timer := time.NewTimer(0)
	defer timer.Stop()

	// Cooling down after a build is important: builds might change files (eg.
	// goimports), and triggering another rebuild in response to a build is
	// dumb.
	cooldown := time.NewTimer(time.Hour)
	cooldown.Stop()
	defer cooldown.Stop()

	buildPending := true

	for {
		select {
		case ev := <-p.evCh:
			ext := filepath.Ext(ev.Path())
			if ext == ".go" && !buildPending {
				buildPending = true
				timer.Reset(time.Millisecond * 50)
			}

		case <-timer.C:
			p.rebuild()
			cooldown.Reset(time.Millisecond * 100)

		case <-cooldown.C:
			buildPending = false

		case err := <-p.app.cmd.err:
			if err == nil {
				err = fmt.Errorf("app exited unexpectedly")
			}

			p.rwmtx.Lock()
			p.err = err
			p.rwmtx.Unlock()

		case <-p.close:
			return
		}
	}
}

func (p *proxy) runWebpack() {
	cmd := command(nodeBin+"/webpack-dev-server",
		"--host", p.webpack.host,
		"--port", p.webpack.port)
	cmd.restart()

	defer cmd.term()

	for {
		select {
		case err := <-cmd.err:
			log.Printf("E: webpack error: %v", err)
			cmd.restart()

		case <-p.close:
			return
		}
	}
}

func (p *proxy) rebuild() {
	log.Println("I: Rebuilding...")
	defer log.Println("I: Rebuild complete")

	p.app.cmd.term()

	p.rwmtx.Lock()
	defer p.rwmtx.Unlock()

	p.goimports()

	cmd := exec.Command("go", "build", "-i", p.AppPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	p.err = err
	if err != nil {
		log.Printf("E: build failed: %v", err)
		return
	}

	p.app.cmd.restart()
	giveUp := time.After(5 * time.Second)

	for {
		conn, err := net.DialTimeout("tcp", p.app.url.Host, time.Millisecond*5)
		if err == nil {
			conn.Close()
			return
		}

		select {
		case err := <-p.app.cmd.err:
			p.app.cmd.err <- err
			return

		case <-giveUp:
			p.app.cmd.term()
			p.app.cmd.err <- fmt.Errorf("app did not come up in a timely manner")
			return

		case <-time.After(time.Millisecond):
			// Loop and try again
		}
	}
}

func (p *proxy) goimports() {
	wg := sync.WaitGroup{}
	wg.Add(len(p.WatchDirs))

	// TODO(astone): if this gets too slow, recursive walk over dirs in
	// WatchDirs and parallelize on those dirs

	for _, dir := range p.WatchDirs {
		go func(dir string) {
			defer wg.Done()

			cmd := exec.Command("goimports", "-w", dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				// Don't set an error: let compile handle all errors
				log.Printf("E: goimports %s: %v\n%s",
					dir, err,
					stringc.Indent(string(out), "    "))
			}
		}(dir)
	}

	wg.Wait()
}

func (p *proxy) serve() error {
	revProxy := httputil.NewSingleHostReverseProxy(p.app.url)

	s := http.Server{
		Addr: p.ProxyAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p.rwmtx.RLock()
			defer p.rwmtx.RUnlock()

			if p.err != nil {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusInternalServerError)

				fmt.Fprintf(w,
					`
					<style>
						body {
							background: #252830;
							color: #fff;
						}
					</style>
					<h1>Error</h1><pre>%s</pre>
					`,
					html.EscapeString(p.err.Error()))

				return
			}

			revProxy.ServeHTTP(w, r)
		}),
	}

	log.Printf("I: Proxying from %s -> %s ...", p.ProxyAddr, p.AppURL)

	return s.ListenAndServe()
}
