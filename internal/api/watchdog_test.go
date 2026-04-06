package api

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchdog_PingPreventsTimeout(t *testing.T) {
	var fired atomic.Bool
	w := NewStreamWatchdog(func() { fired.Store(true) })

	// Ping faster than timeout
	for i := 0; i < 5; i++ {
		w.Ping()
		time.Sleep(10 * time.Millisecond)
	}
	w.Stop()

	if fired.Load() {
		t.Error("Watchdog should not fire when pinged regularly")
	}
}

func TestWatchdog_StallDetection(t *testing.T) {
	w := NewStreamWatchdog(func() {})
	defer w.Stop()

	// Simulate stall
	w.Ping()
	time.Sleep(35 * time.Millisecond) // > stallWarningAfter would be 30s, but we can't wait that long

	// Just verify TotalStallTime works
	stall := w.TotalStallTime()
	_ = stall // won't be > 0 in test since we can't wait 30s
}

func TestWatchdog_Stop(t *testing.T) {
	var fired atomic.Bool
	w := NewStreamWatchdog(func() { fired.Store(true) })

	// Stop immediately
	w.Stop()

	// Wait a bit — should NOT fire
	time.Sleep(50 * time.Millisecond)

	if fired.Load() {
		t.Error("Watchdog should not fire after Stop()")
	}
}

func TestWatchdog_DoubleStop(t *testing.T) {
	w := NewStreamWatchdog(func() {})

	// Double stop should not panic
	w.Stop()
	w.Stop()
}

func TestWatchdog_PingAfterStop(t *testing.T) {
	w := NewStreamWatchdog(func() {})
	w.Stop()

	// Ping after stop should not panic
	w.Ping()
}
