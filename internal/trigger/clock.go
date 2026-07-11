package trigger

import "time"

type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type FakeClock struct {
	now time.Time
}

func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

func (c *FakeClock) Now() time.Time { return c.now }

func (c *FakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- c.now.Add(d)
	return ch
}
