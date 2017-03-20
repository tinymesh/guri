package main

import (
	"fmt"
	"log"
	"time"
)

type Backoff struct {
	initialTime time.Duration
	time        time.Duration
	maxTime     time.Duration
	stepValue   float64
	retries     int
	maxRetries  int
}

// maxRetries := 0 -> no retries
// maxRetries < 0 -> infinite retries
// maxRetries > 0 -> N retries
func NewBackoff(initialTime time.Duration, stepValue float64, maxRetries int) Backoff {
	var maxTime time.Duration

	if 0 == maxRetries {
		maxTime = time.Duration(0)
	} else if maxRetries < 0 {
		maxTime = time.Duration(float64(initialTime) * (1500 * stepValue))
	} else {
		maxTime = time.Duration(float64(initialTime) * (float64(maxRetries) * 100 * stepValue))
	}

	log.Printf("maxTime: %v, maxTries: %v\n", maxTime, maxRetries)

	return Backoff{
		initialTime: initialTime,
		time:        initialTime,
		maxTime:     maxTime,
		stepValue:   stepValue,
		retries:     0,
		maxRetries:  maxRetries,
	}
}

func (backoff *Backoff) Until(call func() error) error {
	for {
		backoff.retries = backoff.retries + 1

		// will fail in case maxRetries := 0 and when retries exceeds maxRetries
		if backoff.maxRetries >= 0 && backoff.retries > backoff.maxRetries {
			return fmt.Errorf("failed after %v (re)tries (max := %v)\n", backoff.retries, backoff.maxRetries)
		}

		err := call()

		if nil != err {
			backoff.Fail()
			time.Sleep(backoff.time)
		} else {
			backoff.Success()
			return nil
		}
	}
}

func (backoff *Backoff) Fail() {
	backoff.time = time.Duration(time.Duration(backoff.stepValue) * backoff.time)

	if backoff.time.Nanoseconds() > backoff.maxTime.Nanoseconds() {
		backoff.time = backoff.maxTime
	}

	time.Sleep(backoff.time)
}

func (backoff *Backoff) Success() {
	backoff.time = backoff.initialTime
	backoff.retries = 0
}
