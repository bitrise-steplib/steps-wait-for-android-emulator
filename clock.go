package main

import "time"

// DefaultClock ...
type DefaultClock struct{}

// Now ...
func (c DefaultClock) Now() time.Time {
	return time.Now()
}

// Since ...
func (c DefaultClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Sleep ...
func (c DefaultClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After ...
func (c DefaultClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
