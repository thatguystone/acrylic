package pool

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/thatguystone/cog/check"
)

func TestPool(t *testing.T) {
	c := check.New(t)

	count := uint64(0)

	var r *Runner
	Pool(&r, func() {
		for i := 0; i < 512; i++ {
			r.Do(func() {
				atomic.AddUint64(&count, 1)
				time.Sleep(time.Microsecond)
			})
		}
	})

	c.Equal(count, 512)
}

func TestRunnerMultiDone(t *testing.T) {
	check.New(t)

	r := NewRunner(1)
	r.Wait()
}
