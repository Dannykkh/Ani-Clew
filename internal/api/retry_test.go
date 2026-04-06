package api

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestClassifyHTTPError(t *testing.T) {
	tests := []struct {
		status   int
		expected ErrorCategory
	}{
		{429, ErrorRateLimit},
		{529, ErrorOverloaded},
		{401, ErrorAuth},
		{403, ErrorAuth},
		{400, ErrorContextOverflow},
		{500, ErrorRetryable},
		{502, ErrorRetryable},
		{503, ErrorRetryable},
		{408, ErrorRetryable},
		{409, ErrorRetryable},
		{200, ErrorFatal},
		{404, ErrorFatal},
	}

	for _, tt := range tests {
		result := ClassifyHTTPError(tt.status, nil)
		if result != tt.expected {
			t.Errorf("ClassifyHTTPError(%d) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestShouldRetry(t *testing.T) {
	cfg := DefaultRetryConfig()

	tests := []struct {
		category ErrorCategory
		attempt  int
		expected bool
	}{
		{ErrorRetryable, 1, true},
		{ErrorRetryable, 10, false}, // max reached
		{ErrorOverloaded, 5, true},
		{ErrorRateLimit, 3, true},
		{ErrorAuth, 1, true},  // try once more
		{ErrorAuth, 2, false}, // no more
		{ErrorFatal, 1, false},
		{ErrorContextOverflow, 1, true},
		{ErrorContextOverflow, 3, false},
	}

	for _, tt := range tests {
		result := ShouldRetry(tt.category, tt.attempt, cfg)
		if result != tt.expected {
			t.Errorf("ShouldRetry(%v, %d) = %v, want %v", tt.category, tt.attempt, result, tt.expected)
		}
	}
}

func TestShouldRetry_Persistent(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.PersistentMode = true

	// In persistent mode, retryable errors always retry
	if !ShouldRetry(ErrorRetryable, 100, cfg) {
		t.Error("Persistent mode should retry even after 100 attempts")
	}

	// But auth errors don't retry
	if ShouldRetry(ErrorAuth, 5, cfg) {
		t.Error("Persistent mode should not retry auth errors")
	}
}

func TestCalculateBackoff(t *testing.T) {
	cfg := DefaultRetryConfig()

	// First attempt: ~500ms base
	d1 := CalculateBackoff(1, cfg, "")
	if d1 < 400*time.Millisecond || d1 > 700*time.Millisecond {
		t.Errorf("Attempt 1 backoff should be ~500ms, got %v", d1)
	}

	// Fourth attempt: ~4s base
	d4 := CalculateBackoff(4, cfg, "")
	if d4 < 3*time.Second || d4 > 6*time.Second {
		t.Errorf("Attempt 4 backoff should be ~4s, got %v", d4)
	}

	// Should cap at MaxDelay
	d20 := CalculateBackoff(20, cfg, "")
	if d20 > cfg.MaxDelay+time.Second {
		t.Errorf("Attempt 20 should cap at %v, got %v", cfg.MaxDelay, d20)
	}
}

func TestCalculateBackoff_RetryAfter(t *testing.T) {
	cfg := DefaultRetryConfig()

	// Server says wait 10 seconds
	d := CalculateBackoff(1, cfg, "10")
	if d != 10*time.Second {
		t.Errorf("Retry-After: 10 should give 10s, got %v", d)
	}

	// Invalid header → fall back to computed
	d2 := CalculateBackoff(1, cfg, "invalid")
	if d2 < 400*time.Millisecond {
		t.Errorf("Invalid Retry-After should fall back, got %v", d2)
	}
}

func TestWithRetry_Success(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.MaxRetries = 3

	callCount := 0
	result := WithRetry(context.Background(), cfg, func() (*http.Response, error) {
		callCount++
		return &http.Response{StatusCode: 200}, nil
	})

	if result.Err != nil {
		t.Errorf("Expected success, got error: %v", result.Err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
	if result.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", result.Attempts)
	}
}

func TestWithRetry_EventualSuccess(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.MaxRetries = 5
	cfg.BaseDelay = 10 * time.Millisecond // fast for tests

	callCount := 0
	result := WithRetry(context.Background(), cfg, func() (*http.Response, error) {
		callCount++
		if callCount < 3 {
			return &http.Response{StatusCode: 500, Body: http.NoBody}, nil
		}
		return &http.Response{StatusCode: 200}, nil
	})

	if result.Err != nil {
		t.Errorf("Expected success after retries, got: %v", result.Err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestClassifyHTTPError_XShouldRetry(t *testing.T) {
	headers := http.Header{}
	headers.Set("x-should-retry", "false")

	// Even a 500 should be fatal if x-should-retry: false
	result := ClassifyHTTPError(500, headers)
	if result != ErrorFatal {
		t.Errorf("x-should-retry: false should make 500 fatal, got %v", result)
	}
}

func TestExtractErrorMessage(t *testing.T) {
	tests := []struct {
		body     string
		expected string
	}{
		{`{"error":{"message":"rate limited"}}`, "rate limited"},
		{`plain text error`, "plain text error"},
		{"", ""},
	}

	for _, tt := range tests {
		result := ExtractErrorMessage([]byte(tt.body))
		if result != tt.expected {
			t.Errorf("ExtractErrorMessage(%q) = %q, want %q", tt.body, result, tt.expected)
		}
	}
}
