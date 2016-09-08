package acrylic

import (
	"os"
	"os/exec"
	"sync"
	"time"
)

// cmd wraps an external command, allowing it to be restarted at will
type cmd struct {
	*exec.Cmd
	err  chan error
	mtx  sync.Mutex
	name string
	args []string
}

func command(name string, args ...string) *cmd {
	return &cmd{
		name: name,
		args: args,
		err:  make(chan error, 1),
	}
}

// Run starts the  command anew
func (c *cmd) restart() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.termUnlocked()

	cmd := exec.Command(c.name, c.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	c.Cmd = cmd

	go func() {
		c.err <- cmd.Run()
	}()
}

func (c *cmd) term() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.termUnlocked()
}

func (c *cmd) termUnlocked() {
	if c.Cmd == nil || c.Cmd.Process == nil {
		c.Cmd = nil
		return
	}

	// Try to be nice
	c.Cmd.Process.Signal(os.Interrupt)

	for {
		// If the process exited somewhere else, we're done here
		if c.Cmd.ProcessState != nil {
			c.Cmd = nil
			return
		}

		select {
		case <-time.After(100 * time.Millisecond):
			c.Cmd.Process.Kill()

		case <-c.err:
			c.Cmd = nil
			return
		}
	}
}
