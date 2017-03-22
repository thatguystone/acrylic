package acrylic

import (
	"runtime"
	"sync"

	"github.com/thatguystone/cog/cync"
)

type bgWork struct {
	mtx  sync.Mutex
	jobs map[string]*bgJob
}

type bgJob struct {
	mtx     sync.Mutex
	cond    sync.Cond
	running bool
	err     error // Last error returned
}

var bgSema = cync.NewSemaphore(runtime.NumCPU() + 1)

// do queues up some background work. If there's already a job running with the
// given key, this does nothing.
func (bg *bgWork) do(key string, cb func() error) {
	bg.mtx.Lock()

	if bg.jobs == nil {
		bg.jobs = map[string]*bgJob{}
	}

	job := bg.jobs[key]
	if job == nil {
		job = new(bgJob)
		job.cond.L = &job.mtx

		bg.jobs[key] = job
	}

	bg.mtx.Unlock()

	job.mtx.Lock()

	if !job.running {
		job.running = true
		go bg.run(job, key, cb)
	}

	job.mtx.Unlock()
}

func (bg *bgWork) run(job *bgJob, key string, cb func() error) {
	var err error
	defer func() {
		if err == nil {
			bg.mtx.Lock()
			delete(bg.jobs, key)
			bg.mtx.Unlock()
		}

		job.mtx.Lock()
		job.running = false
		job.err = err
		job.cond.Broadcast()
		job.mtx.Unlock()
	}()

	bgSema.Lock()
	defer bgSema.Unlock()

	err = cb()
}

func (bg *bgWork) wait(key string) (err error) {
	bg.mtx.Lock()
	job := bg.jobs[key]
	bg.mtx.Unlock()

	if job == nil {
		return
	}

	job.mtx.Lock()

	for job.running {
		job.cond.Wait()
	}

	err = job.err
	job.mtx.Unlock()

	return
}
