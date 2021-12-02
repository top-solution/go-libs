package frequency

import (
	"sync"
	"time"
)

var DefaultScheduler = NewScheduler()

func init() {
	DefaultScheduler.Start()
}

// TaskFn is some task you need to run every so often
type TaskFn func()

// LastRunTimeFn is a func able to fetch the time of the last run
type LastRunTimeFn func() time.Time

// Entry consists of a frequency and the TaskFn to execute on that frequency
type Entry struct {
	Frequency     Frequency
	NextRun       time.Time
	LastRun       time.Time
	TaskFn        TaskFn
	LastRunTimeFn LastRunTimeFn
}

func newEntry(Frequency Frequency) *Entry {
	return &Entry{
		Frequency: Frequency,
		LastRun:   time.Unix(0, 0),
	}
}

// Do adds a TaskFn to the Entry
func (e *Entry) Do(taskFn TaskFn) *Entry {
	e.TaskFn = taskFn
	return e
}

// WithLastRun adds LastRunFn to the Entry
func (e *Entry) WithLastRun(lastRunFn LastRunTimeFn) {
	e.LastRunTimeFn = lastRunFn
}

// Scheduler keeps track of any number of entries, invoking the associated TaskFn
type Scheduler struct {
	entries              []*Entry
	lowResolutionEntries []*Entry
	stop                 chan struct{}
	running              bool

	mu sync.RWMutex
}

// NewScheduler returns a new Scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		entries: nil,
		stop:    make(chan struct{}),
		running: false,
	}
}

// Every schedules a new Entry and returns it.
func (c *Scheduler) Every(frequency Frequency) *Entry {
	entry := newEntry(frequency)

	c.schedule(entry)

	c.mu.Lock()
	defer c.mu.Unlock()

	// The minimum unit is a day: run it with some lazyness
	if frequency.duration == 0 {
		c.lowResolutionEntries = append(c.lowResolutionEntries, entry)
	} else {
		c.entries = append(c.entries, entry)
	}

	return entry
}

// Start the Scheduler in its own go-routine, or no-op if already started.
func (c *Scheduler) Start() {
	if c.running {
		return
	}
	c.running = true
	go c.run()
}

// Stop the Scheduler if it is running.
func (c *Scheduler) Stop() {
	if !c.running {
		return
	}
	c.stop <- struct{}{}
	c.running = false
}

// Stop the Scheduler if it is running, and clear all its task
func (c *Scheduler) Clear() {
	if c.running {
		c.Stop()
	}
	c.entries = make([]*Entry, 0)
	c.lowResolutionEntries = make([]*Entry, 0)
}

func (c *Scheduler) schedule(e *Entry) {
	if e.LastRunTimeFn == nil {
		e.LastRun = time.Now()
	} else {
		e.LastRun = e.LastRunTimeFn()
	}

	e.NextRun = e.Frequency.NextRun(e.LastRun)
}

func (c *Scheduler) run() {
	ticker := time.NewTicker(100 * time.Millisecond)
	lowResolutionTicker := time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				c.runPending(c.entries)
				continue
			case <-c.stop:
				ticker.Stop()
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-lowResolutionTicker.C:
				c.runPending(c.lowResolutionEntries)
				continue
			case <-c.stop:
				lowResolutionTicker.Stop()
				return
			}
		}
	}()
}

func (c *Scheduler) runPending(entries []*Entry) {
	go func() {
		c.mu.RLock()
		for _, entry := range entries {
			// If a custom LastRunTime is given, make sure the schedule is up-to-date
			if entry.LastRunTimeFn != nil {
				c.schedule(entry)
			}
			if time.Now().After(entry.NextRun) {
				go c.runTask(entry)
			}
		}
		c.mu.RUnlock()
	}()
}

func (c *Scheduler) runTask(e *Entry) {
	defer func() {
		if r := recover(); r != nil {
			c.schedule(e)
		}
	}()

	c.schedule(e)
	e.TaskFn()
}
