package acrylic

import (
	"testing"
	"time"

	"github.com/thatguystone/cog/check"
)

type testWatcher struct {
	ch chan WatchEvents
}

func (w testWatcher) Changed(evs WatchEvents) {
	w.ch <- evs
}

func TestWatchBasic(t *testing.T) {
	c := check.New(t)

	fs, clean := c.FS()
	defer clean()

	w := Watch(fs.Path("/"))
	defer w.Stop()

	tw := testWatcher{ch: make(chan WatchEvents, 10)}
	w.Notify(tw)

	select {
	case evs := <-tw.ch:
		c.Len(evs, 0)

	case <-time.After(time.Second):
		c.Fatal("did not get events")
	}

	fs.SWriteFile("/test.ext", "test")

	select {
	case evs := <-tw.ch:
		c.True(evs.HasExt(".ext"))
		c.False(evs.HasExt(".merpmerp"))

	case <-time.After(time.Second):
		c.Fatal("did not get events")
	}
}
