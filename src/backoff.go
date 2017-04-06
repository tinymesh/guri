package main

import "time"

type Backoff struct {
	initial time.Duration
	// data channel
	delay float64
	wait  time.Duration
	// done channel, send something to close it
	max time.Duration
}

func (backoff *Backoff) Fail() {
	time.Sleep(backoff.wait)
	backoff.wait = backoff.wait * time.Duration(backoff.delay)
}

func (backoff *Backoff) Success() {
	backoff.wait = backoff.initial
}
