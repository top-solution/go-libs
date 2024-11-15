package debounce

import (
	"sync"
	"time"
)

func NewWithTimeout(after time.Duration, timeout time.Duration) func(f func()) {
	d := &debouncer{
		after:   after,
		timeout: timeout,
	}

	return func(f func()) {
		d.add(f)
	}
}

type debouncer struct {
	mu       sync.Mutex
	after    time.Duration
	timeout  time.Duration
	timer    *time.Timer
	killTime *time.Timer
	queued   func()
}

func (d *debouncer) add(f func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.queued = f

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.after, d.execute)

	if d.killTime == nil {
		d.killTime = time.AfterFunc(d.timeout, d.execute)
	}
}

func (d *debouncer) execute() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.queued != nil {
		d.queued()
	}

	if d.timer != nil {
		d.timer.Stop()
	}
	if d.killTime != nil {
		d.killTime.Stop()
	}
	d.timer = nil
	d.killTime = nil
	d.queued = nil
}
