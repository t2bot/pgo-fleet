package pgo

import (
	"bytes"
	"io"
	"runtime/pprof"
	"sync"
	"time"
)

var timer *time.Timer
var mu = new(sync.Mutex)

// ErrorFunc is an optional error handler for any errors raised while collecting profiles.
// If not set, errors will be silently ignored (the default).
var ErrorFunc func(err error)

// Endpoint is an interface to describe where to submit profiles to when they're collected.
type Endpoint interface {
	Submit(profile io.Reader) error
}

// Enable begins CPU profiling with the given interval and sampleSize. The first profile will
// begin after 1 interval. After the interval, the CPU profile will begin buffering for sampleSize
// amount of time. Once sampleSize is elapsed, the CPU profile is stopped and submitted to the
// given endpoint.
//
// Only after the profile is fully submitted to the endpoint will another wait for interval begins.
// That is, the time between profiles is `interval + sampleSize + submission time`.
//
// Any encountered errors during the profile and submission will be sent to ErrorFunc.
//
// Enable implies a call to Disable first.
func Enable(interval time.Duration, sampleSize time.Duration, endpoint Endpoint) {
	Disable()

	mu.Lock()
	defer mu.Unlock()

	var localTimer *time.Timer

	reschedule := func() {
		// Don't reschedule if we're the wrong timer
		if timer == localTimer {
			// We use a new goroutine to break free of anything holding us to the timer
			go Enable(interval, sampleSize, endpoint)
		}
	}

	localTimer = time.AfterFunc(interval, func() {
		defer reschedule()

		profile := bytes.NewBuffer(make([]byte, 0))
		err := pprof.StartCPUProfile(profile)
		if err != nil {
			// profile is probably already in progress
			handleError(err)
			return
		}

		sampleTimer := time.NewTimer(sampleSize)
		defer sampleTimer.Stop()
		<-sampleTimer.C

		pprof.StopCPUProfile() // blocks until write is complete

		// Submit the profile
		err = endpoint.Submit(profile)
		if err != nil {
			handleError(err)
		}
	})
	timer = localTimer
}

// Disable prevents the next CPU profile from being run, but if a profile is already in progress
// then that profile will be submitted. That profile runner will not be rescheduled, however.
func Disable() {
	mu.Lock()
	defer mu.Unlock()
	if timer != nil {
		timer.Stop()
	}
	timer = nil
}

func handleError(err error) {
	if ErrorFunc != nil {
		ErrorFunc(err)
	}
}
