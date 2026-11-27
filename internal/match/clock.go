package match

import (
	"sync"
	"time"
)

// Clock abstracts time for the match loop (inject FakeClock in tests).
type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Timer
}

// Timer is a cancelable delayed callback.
type Timer interface {
	Stop() bool
}

// RealClock uses the process wall clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

func (RealClock) AfterFunc(d time.Duration, f func()) Timer {
	return realTimer{t: time.AfterFunc(d, f)}
}

type realTimer struct{ t *time.Timer }

func (r realTimer) Stop() bool { return r.t.Stop() }

// FakeClock is a controllable clock for tests. Callbacks run on Advance.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
	seq    int
}

func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	t := &fakeTimer{
		clock: c,
		when:  c.now.Add(d),
		fn:    f,
		id:    c.seq,
	}
	c.timers = append(c.timers, t)
	return t
}

// Advance moves time forward and fires due timers in order (re-entrant safe).
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	target := c.now.Add(d)
	c.mu.Unlock()

	for {
		c.mu.Lock()
		var next *fakeTimer
		for _, t := range c.timers {
			if t.stopped || t.fired {
				continue
			}
			if t.when.After(target) {
				continue
			}
			if next == nil || t.when.Before(next.when) || (t.when.Equal(next.when) && t.id < next.id) {
				next = t
			}
		}
		if next == nil {
			c.now = target
			c.mu.Unlock()
			return
		}
		c.now = next.when
		next.fired = true
		fn := next.fn
		c.mu.Unlock()
		fn()
	}
}

type fakeTimer struct {
	clock   *FakeClock
	when    time.Time
	fn      func()
	id      int
	stopped bool
	fired   bool
}

func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if t.stopped || t.fired {
		return false
	}
	t.stopped = true
	return true
}
