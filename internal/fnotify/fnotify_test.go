package fnotify

import (
	"os"
	"testing"
	"time"

	"github.com/thatguystone/cog/check"
)

func TestMain(m *testing.M) {
	check.Main(m)
}

func TestBasic(t *testing.T) {
	c := check.New(t)

	n := New()
	defer n.Exit()

	drain := func() {
		for len(n.C) > 0 {
			<-n.C
		}
	}

	path := c.FS.Path("test")
	n.AddDir(c.FS.Path(""))

	c.FS.SWriteFile("test", "test")
	name := <-n.C
	c.Equal(name, path)
	drain()

	path = c.FS.Path("sub/sub/sub/dirs")
	err := os.MkdirAll(path, 0750)
	c.MustNotError(err)

	select {
	case <-n.C:
	case <-time.After(time.Second):
		c.Fatal("did not get add event")
	}

	drain()
	err = os.RemoveAll(path)
	c.MustNotError(err)

	for {
		select {
		case name = <-n.C:
			if name == path {
				return
			}

		case <-time.After(time.Second):
			c.Fatal("did not get remove event")
		}
	}
}
