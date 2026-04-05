package api

import (
	"log"
	"sync"
	"time"
)

const (
	defaultIdleTimeout = 90 * time.Second  // abort if no events for 90s
	stallWarningAfter  = 30 * time.Second  // log warning after 30s gap
)

// StreamWatchdog monitors an SSE stream for idle/stall conditions.
type StreamWatchdog struct {
	mu          sync.Mutex
	timer       *time.Timer
	lastEvent   time.Time
	stallStart  time.Time
	totalStall  time.Duration
	onTimeout   func() // called when stream is considered dead
	idleTimeout time.Duration
	stopped     bool
}

// NewStreamWatchdog creates a watchdog that calls onTimeout if no events arrive.
func NewStreamWatchdog(onTimeout func()) *StreamWatchdog {
	w := &StreamWatchdog{
		lastEvent:   time.Now(),
		onTimeout:   onTimeout,
		idleTimeout: defaultIdleTimeout,
	}
	w.timer = time.AfterFunc(w.idleTimeout, w.fire)
	return w
}

// Ping resets the watchdog timer. Call on each SSE event.
func (w *StreamWatchdog) Ping() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}

	now := time.Now()
	gap := now.Sub(w.lastEvent)

	// Detect stall (>30s gap between events)
	if gap > stallWarningAfter {
		if w.stallStart.IsZero() {
			w.stallStart = w.lastEvent
		}
		stallDuration := now.Sub(w.stallStart)
		w.totalStall += gap
		log.Printf("[StreamWatchdog] Stall detected: %.1fs gap (total stall: %.1fs)", gap.Seconds(), stallDuration.Seconds())
	} else {
		w.stallStart = time.Time{}
	}

	w.lastEvent = now

	// Reset timer
	if !w.timer.Stop() {
		// Timer already fired — drain channel if buffered
		select {
		case <-w.timer.C:
		default:
		}
	}
	w.timer.Reset(w.idleTimeout)
}

// Stop disarms the watchdog. Call when stream completes normally.
func (w *StreamWatchdog) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stopped = true
	w.timer.Stop()
}

// TotalStallTime returns cumulative time spent in stall state.
func (w *StreamWatchdog) TotalStallTime() time.Duration {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.totalStall
}

func (w *StreamWatchdog) fire() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	cb := w.onTimeout
	w.mu.Unlock()

	log.Printf("[StreamWatchdog] Stream idle timeout (%.0fs). Aborting.", w.idleTimeout.Seconds())
	if cb != nil {
		cb()
	}
}
