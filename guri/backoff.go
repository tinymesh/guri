package guri

import "time"

// Backoff ...
type Backoff struct {
	initial time.Duration
	// data channel
	delay float64
	wait  time.Duration
	// done channel, send something to close it
	max time.Duration
}

// Fail mark attempt as failed, increases backoff timer
func (backoff *Backoff) Fail() {
	time.Sleep(backoff.wait)
	backoff.wait = backoff.wait * time.Duration(backoff.delay)
}

// Success mark attemp as successfull
func (backoff *Backoff) Success() {
	backoff.wait = backoff.initial
}
