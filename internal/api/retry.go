package api

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RetryConfig controls retry behavior.
type RetryConfig struct {
	MaxRetries      int           // max attempts (default 10)
	BaseDelay       time.Duration // initial delay (default 500ms)
	MaxDelay        time.Duration // cap delay (default 32s)
	PersistentMode  bool          // for unattended: infinite retries, up to 5min delay
}

// DefaultRetryConfig returns standard retry settings.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 10,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   32 * time.Second,
	}
}

// RetryResult holds the outcome of a retry-wrapped operation.
type RetryResult struct {
	Response    *http.Response
	Err         error
	Attempts    int
	TotalDelay  time.Duration
	FinalStatus int
	UsedFallback bool
}

// ErrorCategory classifies an error for retry decisions.
type ErrorCategory int

const (
	ErrorRetryable   ErrorCategory = iota // transient, retry with backoff
	ErrorFatal                             // don't retry
	ErrorRateLimit                         // 429, special handling
	ErrorOverloaded                        // 529, aggressive retry with fallback
	ErrorAuth                              // 401/403, refresh credentials
	ErrorContextOverflow                   // 400, adjust max_tokens
)

// ClassifyHTTPError determines the error category from status code and headers.
func ClassifyHTTPError(statusCode int, headers http.Header) ErrorCategory {
	// x-should-retry header takes precedence
	if headers != nil {
		if shouldRetry := headers.Get("x-should-retry"); shouldRetry == "false" {
			return ErrorFatal // server explicitly says don't retry
		}
	}
	switch {
	case statusCode == 429:
		return ErrorRateLimit
	case statusCode == 529:
		return ErrorOverloaded
	case statusCode == 401 || statusCode == 403:
		return ErrorAuth
	case statusCode == 400:
		return ErrorContextOverflow
	case statusCode == 408 || statusCode == 409:
		return ErrorRetryable
	case statusCode >= 500:
		return ErrorRetryable
	default:
		return ErrorFatal
	}
}

// ShouldRetry decides if an error should be retried based on category and context.
func ShouldRetry(category ErrorCategory, attempt int, cfg RetryConfig) bool {
	if cfg.PersistentMode {
		// In persistent mode, retry everything except auth errors
		return category != ErrorAuth && category != ErrorFatal
	}

	switch category {
	case ErrorRetryable, ErrorOverloaded:
		return attempt < cfg.MaxRetries
	case ErrorRateLimit:
		// Check if retry-after is reasonable
		return attempt < cfg.MaxRetries
	case ErrorAuth:
		return attempt < 2 // try once more (credential refresh)
	case ErrorContextOverflow:
		return attempt < 3 // try with adjusted params
	default:
		return false
	}
}

// CalculateBackoff computes the delay for the next retry attempt.
// Uses exponential backoff with jitter to prevent thundering herd.
func CalculateBackoff(attempt int, cfg RetryConfig, retryAfterHeader string) time.Duration {
	// Server-specified retry-after takes precedence
	if retryAfterHeader != "" {
		if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
			serverDelay := time.Duration(seconds) * time.Second
			if serverDelay > 0 && serverDelay < 5*time.Minute {
				return serverDelay
			}
		}
	}

	// Exponential backoff: base * 2^(attempt-1)
	base := float64(cfg.BaseDelay)
	delay := base * math.Pow(2, float64(attempt-1))

	// Cap at max delay
	maxDelay := float64(cfg.MaxDelay)
	if cfg.PersistentMode {
		maxDelay = float64(5 * time.Minute)
	}
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter: 0-25% of base delay
	jitter := rand.Float64() * 0.25 * base
	delay += jitter

	return time.Duration(delay)
}

// WithRetry wraps an HTTP request function with retry logic.
func WithRetry(ctx context.Context, cfg RetryConfig, doRequest func() (*http.Response, error)) RetryResult {
	result := RetryResult{}
	var lastErr error
	var lastStatus int

	for attempt := 1; attempt <= cfg.MaxRetries || cfg.PersistentMode; attempt++ {
		result.Attempts = attempt

		// Check context cancellation
		if ctx.Err() != nil {
			result.Err = ctx.Err()
			return result
		}

		resp, err := doRequest()
		if err != nil {
			// Connection error — always retryable
			lastErr = err
			delay := CalculateBackoff(attempt, cfg, "")
			log.Printf("[API Retry] Connection error (attempt %d): %v. Retrying in %v", attempt, err, delay)
			result.TotalDelay += delay
			sleepWithContext(ctx, delay)
			continue
		}

		lastStatus = resp.StatusCode
		result.FinalStatus = lastStatus

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Response = resp
			return result
		}

		// Classify the error
		category := ClassifyHTTPError(resp.StatusCode, resp.Header)
		resp.Body.Close()

		if !ShouldRetry(category, attempt, cfg) {
			result.Err = fmt.Errorf("HTTP %d (non-retryable)", resp.StatusCode)
			return result
		}

		// Calculate delay
		retryAfter := resp.Header.Get("Retry-After")
		delay := CalculateBackoff(attempt, cfg, retryAfter)

		log.Printf("[API Retry] HTTP %d (%s) attempt %d/%d. Retrying in %v",
			resp.StatusCode, categoryName(category), attempt, cfg.MaxRetries, delay)

		result.TotalDelay += delay
		sleepWithContext(ctx, delay)
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	result.Err = fmt.Errorf("max retries exceeded (last: %v, status: %d)", lastErr, lastStatus)
	return result
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func categoryName(c ErrorCategory) string {
	switch c {
	case ErrorRetryable:
		return "retryable"
	case ErrorFatal:
		return "fatal"
	case ErrorRateLimit:
		return "rate-limit"
	case ErrorOverloaded:
		return "overloaded"
	case ErrorAuth:
		return "auth"
	case ErrorContextOverflow:
		return "context-overflow"
	default:
		return "unknown"
	}
}

// ExtractErrorMessage tries to extract a meaningful error message from an HTTP response body.
func ExtractErrorMessage(body []byte) string {
	s := string(body)
	// Try Anthropic format: {"error":{"message":"..."}}
	if idx := strings.Index(s, `"message":"`); idx >= 0 {
		start := idx + len(`"message":"`)
		end := strings.Index(s[start:], `"`)
		if end > 0 {
			return s[start : start+end]
		}
	}
	// Truncate raw body
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
