package acrylic

import (
	"os"
	"os/exec"
	"sync"
	"time"
)

// Cmd wraps a command, allowing it to be restarted at will
type Cmd struct {
	*exec.Cmd
	Err  chan error
	mtx  sync.Mutex
	name string
	args []string
}

func Command(name string, args ...string) *Cmd {
	return &Cmd{
		name: name,
		args: args,
		Err:  make(chan error, 1),
	}
}

// Run starts the  command anew
func (a *Cmd) Restart() {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	a.term()

	cmd := exec.Command(a.name, a.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	a.Cmd = cmd

	go func() {
		a.Err <- cmd.Run()
	}()
}

func (a *Cmd) Term() {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.term()
}

func (a *Cmd) term() {
	if a.Cmd == nil || a.Cmd.Process == nil {
		a.Cmd = nil
		return
	}

	// Try to be nice
	a.Cmd.Process.Signal(os.Interrupt)

	for {
		// If the process exited somewhere else, we're done here
		if a.Cmd.ProcessState != nil {
			a.Cmd = nil
			return
		}

		select {
		case <-time.After(100 * time.Millisecond):
			a.Cmd.Process.Kill()

		case <-a.Err:
			a.Cmd = nil
			return
		}
	}
}
