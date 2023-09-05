package pgo

import (
	"bytes"
	"errors"
	"io"
	"runtime/pprof"
	"testing"
	"time"
)

type testEndpoint struct {
	Endpoint
	calls int
	throw error
}

func (t *testEndpoint) Submit(profile io.Reader) error {
	if profile == nil {
		panic("expected reader")
	}
	t.calls += 1
	return t.throw
}

func TestDisableNoTimer(t *testing.T) {
	timer = nil
	Disable() // we're testing that it doesn't panic
	if timer != nil {
		t.Fatal("Expected nil timer after disable, got not-nil")
	}
}

func TestEnable(t *testing.T) {
	testTimer := time.NewTimer(5 * time.Second)
	defer testTimer.Stop()
	timer = testTimer

	endpoint := &testEndpoint{}
	Enable(1*time.Second, 1*time.Second, endpoint)
	if timer == testTimer {
		t.Fatal("Expected timer to be replaced, got same")
	}
	if timer == nil {
		t.Fatal("Expected timer to be set, got nil")
	}

	firstTest := true
	defer pprof.StopCPUProfile()
retest:

	// Wait for the timer to start
	<-time.After(1100 * time.Millisecond)

	// We should now get an error if we start profiling
	err := pprof.StartCPUProfile(bytes.NewBuffer(make([]byte, 0)))
	if err == nil {
		t.Fatalf("Expected pprof profile to have started. firstRun=%t , got nil", firstTest)
	}

	// Wait for profile to conclude
	<-time.After(1100 * time.Millisecond)

	// We should have received the profile
	if endpoint.calls != 1 {
		t.Fatalf("Expected 1 profile to be submitted. firstRun=%t , got %d", firstTest, endpoint.calls)
	}
	err = pprof.StartCPUProfile(bytes.NewBuffer(make([]byte, 0)))
	if err != nil {
		t.Fatalf("Expected pprof profile to have finished (no error). firstRun=%t , got '%s'", firstTest, err.Error())
	}

	// Cleanup and retest if needed
	pprof.StopCPUProfile()
	if firstTest {
		firstTest = false
		endpoint.calls = 0
		goto retest
	}

	// Final cleanup
	Disable()
}

func TestProfileInProgress(t *testing.T) {
	endpoint := &testEndpoint{}
	Enable(1*time.Second, 1*time.Second, endpoint)
	defer pprof.StopCPUProfile()
	err := pprof.StartCPUProfile(bytes.NewBuffer(make([]byte, 0)))
	if err != nil {
		t.Fatalf("Expected no error when starting CPU profile, got '%s'", err.Error())
	}
	<-time.After(2100 * time.Millisecond)
	if endpoint.calls != 0 {
		t.Fatalf("Expected 0 calls (no profile possible), got %d", endpoint.calls)
	}
	Disable() // cleanup
}

func TestWriteError(t *testing.T) {
	endpoint := &testEndpoint{
		throw: errors.New("thrown"),
	}
	Enable(1*time.Second, 1*time.Second, endpoint)
	defer pprof.StopCPUProfile() // just in case
	<-time.After(2100 * time.Millisecond)
	if endpoint.calls != 1 {
		t.Fatalf("Expected 1 call, got %d", endpoint.calls)
	}
	Disable() // cleanup
}

func TestErrorFunc(t *testing.T) {
	endpoint := &testEndpoint{
		throw: errors.New("thrown"),
	}
	errCalls := 0
	errFn := func(err error) {
		errCalls += 1
		if err == nil {
			panic("expected error")
		}
		if errCalls == 1 {
			// XXX: Magic string being tested here.
			if err.Error() != "cpu profiling already in use" {
				panic("wrong error encountered: " + err.Error())
			}
		} else if errCalls == 2 {
			//goland:noinspection GoDirectComparisonOfErrors
			if err != endpoint.throw {
				panic("wrong error encountered: " + err.Error())
			}
		}
	}
	ErrorFunc = errFn
	defer func() {
		ErrorFunc = nil
	}()

	// First test: Does having pprof started elsewhere cause an error?
	Enable(1*time.Second, 1*time.Second, endpoint)
	defer pprof.StopCPUProfile()
	err := pprof.StartCPUProfile(bytes.NewBuffer(make([]byte, 0)))
	if err != nil {
		t.Fatalf("Expected no error when starting CPU profile, got '%s'", err.Error())
	}
	<-time.After(1600 * time.Millisecond) // ensure we land after the pprof start, but before the second run
	Disable()                             // prevent that second run
	if endpoint.calls != 0 {
		t.Fatalf("Expected 0 calls (no profile possible), got %d", endpoint.calls)
	}
	if errCalls != 1 {
		t.Fatalf("Expected 1 error call, got %d", errCalls)
	}
	pprof.StopCPUProfile() // cleanup before next test

	// Second test: What about a write error?
	Enable(1*time.Second, 1*time.Second, endpoint)
	<-time.After(2100 * time.Millisecond)
	if endpoint.calls != 1 {
		t.Fatalf("Expected 1 call, got %d", endpoint.calls)
	}
	Disable() // cleanup
	if errCalls != 2 {
		t.Fatalf("Expected 2 error calls, got %d", errCalls)
	}
}

func TestDoubleEnable(t *testing.T) {
	endpoint1 := &testEndpoint{}
	endpoint2 := &testEndpoint{}
	Enable(1*time.Second, 1*time.Second, endpoint1)
	Enable(1*time.Second, 1*time.Second, endpoint2)
	<-time.After(2100 * time.Millisecond)
	Disable() // cleanup now (because why not)
	if endpoint1.calls != 0 {
		t.Fatalf("Expected 0 calls to first endpoint, got %d", endpoint1.calls)
	}
	if endpoint2.calls != 1 {
		t.Fatalf("Expected 1 calls to second endpoint, got %d", endpoint2.calls)
	}
}
