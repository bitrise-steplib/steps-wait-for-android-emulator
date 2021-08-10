package main

import "time"

type defaultClock struct{}

// Now ...
func (c defaultClock) Now() time.Time {
	return time.Now()
}

// Since ...
func (c defaultClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Sleep ...
func (c defaultClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After ...
func (c defaultClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
