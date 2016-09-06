package pool

import (
	"runtime"
	"sync"
)

// A Runner is for running jobs concurrently
type Runner struct {
	wg     sync.WaitGroup
	workWg sync.WaitGroup
	work   chan func()
}

// Pool executes a new pool of size runtime.GOMAXPROCS(-1) and calls the
// callback with the pool.
//
// If wait is true, this waits for all jobs to finish before returning.
func Pool(r **Runner, cb func()) {
	tr := NewRunner(runtime.GOMAXPROCS(-1))
	*r = tr
	cb()
	tr.Wait()
	*r = nil
}

// NewRunner creates a new Runner
func NewRunner(size int) *Runner {
	r := &Runner{
		work: make(chan func(), size*16),
	}

	r.wg.Add(size)
	for i := 0; i < size; i++ {
		go r.worker()
	}

	return r
}

func (r *Runner) worker() {
	defer r.wg.Done()

	for w := range r.work {
		w()
		r.workWg.Done()
	}
}

// Do submits some work to the pool
func (r *Runner) Do(work func()) {
	r.workWg.Add(1)
	r.work <- work
}

// Wait waits for everyone to be done
func (r *Runner) Wait() {
	r.workWg.Wait()
	close(r.work)
	r.wg.Wait()
}
