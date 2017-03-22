package acrylic

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/thatguystone/cog/check"
)

func TestBGWorkBasic(t *testing.T) {
	c := check.New(t)
	bg := new(bgWork)

	wg := sync.WaitGroup{}
	wg.Add(1)

	var runs uint32
	for i := 0; i < 10; i++ {
		bg.do("a", func() error {
			wg.Wait()
			atomic.AddUint32(&runs, 1)
			return errors.New("persistent error")
		})
	}

	wg.Done()

	for i := 0; i < 5; i++ {
		err := bg.wait("a")
		c.Must.NotNil(err)
		c.Equal(err.Error(), "persistent error")
	}

	c.Equal(runs, 1)

	bg.do("a", func() error { return nil })
	c.Nil(bg.wait("a"))
	c.Len(bg.jobs, 0)
	c.Nil(bg.wait("a"))
}
