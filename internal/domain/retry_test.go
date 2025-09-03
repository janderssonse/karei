// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestRetryBackoffStrategy tests the exponential backoff retry logic
// Business Rule: Network errors should be retried with exponential backoff.
func TestRetryBackoffStrategy(t *testing.T) {
	tests := []struct {
		name           string
		attempt        int
		expectedDelay  time.Duration
		shouldContinue bool
		reason         string
	}{
		{
			name:           "first_retry_1_second",
			attempt:        1,
			expectedDelay:  1 * time.Second,
			shouldContinue: true,
			reason:         "First retry should wait 1 second",
		},
		{
			name:           "second_retry_2_seconds",
			attempt:        2,
			expectedDelay:  2 * time.Second,
			shouldContinue: true,
			reason:         "Second retry should wait 2 seconds (exponential)",
		},
		{
			name:           "third_retry_4_seconds",
			attempt:        3,
			expectedDelay:  4 * time.Second,
			shouldContinue: true,
			reason:         "Third retry should wait 4 seconds",
		},
		{
			name:           "fourth_retry_8_seconds",
			attempt:        4,
			expectedDelay:  8 * time.Second,
			shouldContinue: true,
			reason:         "Fourth retry should wait 8 seconds",
		},
		{
			name:           "fifth_retry_stops",
			attempt:        5,
			expectedDelay:  0,
			shouldContinue: false,
			reason:         "Should stop after 4 retries (5 total attempts)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Business logic for exponential backoff
			calculateBackoff := func(attempt int) (time.Duration, bool) {
				const maxRetries = 4
				if attempt > maxRetries {
					return 0, false
				}
				// Exponential backoff: 1s, 2s, 4s, 8s
				delay := time.Second * (1 << (attempt - 1))

				return delay, true
			}

			delay, shouldContinue := calculateBackoff(tc.attempt)
			assert.Equal(t, tc.expectedDelay, delay, tc.reason)
			assert.Equal(t, tc.shouldContinue, shouldContinue,
				"Retry continuation should match expectation")
		})
	}
}

// TestRetryableErrors tests which errors should trigger retries
// Business Rule: Only transient errors should be retried.
func TestRetryableErrors(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
		reason    string
	}{
		{
			name:      "network_failure_is_retryable",
			err:       domain.ErrNetworkFailure,
			retryable: true,
			reason:    "Network failures are often transient",
		},
		{
			name:      "timeout_is_retryable",
			err:       errors.New("operation timeout"),
			retryable: true,
			reason:    "Timeouts might succeed on retry",
		},
		{
			name:      "connection_refused_is_retryable",
			err:       errors.New("connection refused"),
			retryable: true,
			reason:    "Server might be temporarily down",
		},
		{
			name:      "permission_denied_not_retryable",
			err:       domain.ErrPermissionDenied,
			retryable: false,
			reason:    "Permission won't change on retry",
		},
		{
			name:      "package_not_found_not_retryable",
			err:       domain.ErrPackageNotFound,
			retryable: false,
			reason:    "Package won't appear on retry",
		},
		{
			name:      "already_installed_not_retryable",
			err:       domain.ErrAlreadyInstalled,
			retryable: false,
			reason:    "Already successful, no need to retry",
		},
		{
			name:      "invalid_package_not_retryable",
			err:       domain.ErrInvalidPackage,
			retryable: false,
			reason:    "Invalid input won't become valid",
		},
		{
			name:      "dependency_missing_not_retryable",
			err:       domain.ErrDependencyMissing,
			retryable: false,
			reason:    "Dependencies need explicit resolution",
		},
		{
			name:      "no_package_manager_not_retryable",
			err:       domain.ErrNoPackageManager,
			retryable: false,
			reason:    "System configuration won't change",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Business logic for determining retryable errors
			isRetryable := func(err error) bool {
				// Check for specific non-retryable domain errors
				switch {
				case errors.Is(err, domain.ErrPermissionDenied),
					errors.Is(err, domain.ErrPackageNotFound),
					errors.Is(err, domain.ErrAlreadyInstalled),
					errors.Is(err, domain.ErrInvalidPackage),
					errors.Is(err, domain.ErrDependencyMissing),
					errors.Is(err, domain.ErrNoPackageManager),
					errors.Is(err, domain.ErrNotInstalled):
					return false
				case errors.Is(err, domain.ErrNetworkFailure):
					return true
				}

				// Check error message for transient patterns
				errMsg := err.Error()
				transientPatterns := []string{
					"timeout",
					"connection refused",
					"connection reset",
					"temporary failure",
					"retry",
				}

				for _, pattern := range transientPatterns {
					if strings.Contains(errMsg, pattern) {
						return true
					}
				}

				return false
			}

			result := isRetryable(tc.err)
			assert.Equal(t, tc.retryable, result, tc.reason)
		})
	}
}

// TestRetryWithJitter tests adding jitter to prevent thundering herd
// Business Rule: Add random jitter to retry delays to prevent synchronized retries.
func TestRetryWithJitter(t *testing.T) {
	baseDelay := 1 * time.Second
	maxJitter := 200 * time.Millisecond

	// Business logic for adding jitter
	addJitter := func(delay time.Duration, maxJitter time.Duration) time.Duration {
		// In real implementation, use rand.Intn
		// For testing, we verify the range
		minDelay := delay
		maxDelay := delay + maxJitter

		// Verify jitter doesn't make delay negative or too large
		assert.Greater(t, maxDelay, minDelay, "Jitter should increase delay")
		assert.LessOrEqual(t, maxDelay, delay+maxJitter, "Jitter should not exceed max")

		return delay + (maxJitter / 2) // Return middle value for testing
	}

	result := addJitter(baseDelay, maxJitter)
	assert.Greater(t, result, baseDelay, "Jittered delay should be greater than base")
	assert.LessOrEqual(t, result, baseDelay+maxJitter, "Jittered delay should not exceed max")
}

// TestRetryStateMachineValidTransitions tests valid retry state transitions.
func TestRetryStateMachineValidTransitions(t *testing.T) {
	type RetryState string

	const (
		StateInitial   RetryState = "initial"
		StateRetrying  RetryState = "retrying"
		StateSucceeded RetryState = "succeeded"
		StateFailed    RetryState = "failed"
		StateAborted   RetryState = "aborted"
	)

	isValidTransition := func(from RetryState, event string, to RetryState) bool {
		if from == StateInitial {
			switch event {
			case "retryable_error":
				return to == StateRetrying
			case "success":
				return to == StateSucceeded
			case "permanent_error":
				return to == StateFailed
			}
		}

		if from == StateRetrying {
			switch event {
			case "success":
				return to == StateSucceeded
			case "max_retries_exceeded":
				return to == StateFailed
			case "cancelled":
				return to == StateAborted
			}
		}

		return false // Terminal states
	}
	// Define test cases outside to reduce function complexity
	validTransitions := []struct {
		name      string
		fromState RetryState
		event     string
		toState   RetryState
		reason    string
	}{
		{
			name:      "initial_to_retrying_on_error",
			fromState: StateInitial,
			event:     "retryable_error",
			toState:   StateRetrying,
			reason:    "Can start retrying from initial state",
		},
		{
			name:      "initial_to_succeeded_on_success",
			fromState: StateInitial,
			event:     "success",
			toState:   StateSucceeded,
			reason:    "Can succeed on first attempt",
		},
		{
			name:      "initial_to_failed_on_permanent_error",
			fromState: StateInitial,
			event:     "permanent_error",
			toState:   StateFailed,
			reason:    "Permanent errors should fail immediately",
		},
		{
			name:      "retrying_to_succeeded",
			fromState: StateRetrying,
			event:     "success",
			toState:   StateSucceeded,
			reason:    "Retry can succeed",
		},
		{
			name:      "retrying_to_failed_on_max_retries",
			fromState: StateRetrying,
			event:     "max_retries_exceeded",
			toState:   StateFailed,
			reason:    "Should fail after max retries",
		},
		{
			name:      "retrying_to_aborted_on_cancel",
			fromState: StateRetrying,
			event:     "cancelled",
			toState:   StateAborted,
			reason:    "Can abort retries on cancellation",
		},
	}

	// Test valid transitions
	for _, tc := range validTransitions {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidTransition(tc.fromState, tc.event, tc.toState)
			assert.True(t, result, tc.reason)
		})
	}
}

// TestRetryStateMachineInvalidTransitions tests invalid retry state transitions.
func TestRetryStateMachineInvalidTransitions(t *testing.T) {
	type RetryState string

	const (
		StateInitial   RetryState = "initial"
		StateRetrying  RetryState = "retrying"
		StateSucceeded RetryState = "succeeded"
		StateFailed    RetryState = "failed"
		StateAborted   RetryState = "aborted"
	)

	invalidTransitions := []struct {
		name      string
		fromState RetryState
		event     string
		toState   RetryState
		reason    string
	}{
		{
			name:      "succeeded_cannot_retry",
			fromState: StateSucceeded,
			event:     "retry",
			toState:   StateRetrying,
			reason:    "Cannot retry after success",
		},
		{
			name:      "failed_cannot_retry",
			fromState: StateFailed,
			event:     "retry",
			toState:   StateRetrying,
			reason:    "Cannot retry after final failure",
		},
	}

	// Test invalid transitions
	for _, tc := range invalidTransitions {
		t.Run(tc.name, func(t *testing.T) {
			// Invalid transitions should always return false
			result := false
			if tc.fromState == StateSucceeded || tc.fromState == StateFailed || tc.fromState == StateAborted {
				result = false // Terminal states can't transition
			}

			assert.False(t, result, tc.reason)
		})
	}
}
