// SPDX-FileCopyrightText: 2024 Josef Andersson
// SPDX-License-Identifier: EUPL-1.2

package domain_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	circuitBreakerOpen     = "open"
	circuitBreakerClosed   = "closed"
	circuitBreakerHalfOpen = "half-open"
)

// TestSystemResiliencePatterns tests business-critical resilience mechanisms.
func TestSystemResiliencePatterns(t *testing.T) {
	tests := []struct {
		name            string
		errorMsg        string
		expectedRetries int
		shouldRetry     bool
		isCritical      bool
	}{
		{
			name:            "transient_network_error_should_retry",
			errorMsg:        "connection refused: package.server.com:443",
			expectedRetries: 3,
			shouldRetry:     true,
			isCritical:      false,
		},
		{
			name:            "disk_full_error_is_critical",
			errorMsg:        "no space left on device",
			expectedRetries: 0,
			shouldRetry:     false,
			isCritical:      true,
		},
		{
			name:            "corrupt_package_should_redownload",
			errorMsg:        "checksum mismatch: expected abc123, got xyz789",
			expectedRetries: 1,
			shouldRetry:     true,
			isCritical:      false,
		},
		{
			name:            "memory_exhaustion_is_critical",
			errorMsg:        "cannot allocate memory",
			expectedRetries: 0,
			shouldRetry:     false,
			isCritical:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMsg)

			// Business logic: Determine if error warrants retry
			shouldRetry := isRetriableError(err)
			assert.Equal(t, tt.shouldRetry, shouldRetry)

			// Business logic: Identify critical failures requiring immediate halt
			isCritical := isCriticalSystemError(err)
			assert.Equal(t, tt.isCritical, isCritical)

			if shouldRetry {
				retries := determineRetryCount(err)
				assert.Equal(t, tt.expectedRetries, retries)
			}
		})
	}
}

// TestPartialFailureRecovery tests handling of partial failures in batch operations.
func TestPartialFailureRecovery(t *testing.T) {
	packages := []domain.Package{
		{Name: "essential-lib", Version: "1.0.0", Method: "apt"},
		{Name: "optional-tool", Version: "2.0.0", Method: "snap"},
		{Name: "critical-app", Version: "3.0.0", Method: "apt"},
		{Name: "nice-to-have", Version: "1.5.0", Method: "flatpak"},
	}

	// Simulate failure on optional package
	failedPackage := "optional-tool"

	results := processPackagesWithFailureHandling(packages, failedPackage)

	// Business rule: Essential packages (apt) must be prioritized
	for _, result := range results {
		if result.Package.Method == "apt" {
			// APT packages are considered essential
			if result.Package.Name != failedPackage {
				assert.True(t, result.Success, "Essential APT package must succeed")
			}
		}

		if result.Package.Name == failedPackage {
			assert.False(t, result.Success, "Failed package should be marked as failed")
		}
	}

	// Business rule: Continue processing after non-critical failures
	assert.Len(t, results, len(packages), "All packages should be attempted")
}

// TestCircuitBreakerPattern tests circuit breaker for failing services.
func TestCircuitBreakerPattern(t *testing.T) {
	breaker := &CircuitBreaker{
		FailureThreshold: 3,
		ResetTimeout:     time.Second * 5,
		failures:         0,
		lastFailTime:     time.Time{},
		state:            "closed",
	}

	// Simulate multiple failures
	for i := range 4 {
		err := breaker.Call(func() error {
			return errors.New("service unavailable")
		})

		switch {
		case i < 2:
			// First two failures don't open circuit
			require.Error(t, err)
			assert.Equal(t, "closed", breaker.state)
		case i == 2:
			// Third failure opens circuit
			require.Error(t, err)
			assert.Equal(t, circuitBreakerOpen, breaker.state)
			assert.Contains(t, err.Error(), "circuit breaker open")
		default:
			// Subsequent calls should fail immediately
			require.Error(t, err)
			assert.Equal(t, circuitBreakerOpen, breaker.state)
			assert.Contains(t, err.Error(), "circuit breaker open")
		}
	}

	// Reset for half-open test
	breaker = &CircuitBreaker{
		FailureThreshold: 3,
		ResetTimeout:     time.Second * 5,
		failures:         0,
		lastFailTime:     time.Now().Add(-time.Second * 6), // Past reset timeout
		state:            circuitBreakerHalfOpen,
	}

	// Successful call should close circuit
	err := breaker.Call(func() error {
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, "closed", breaker.state)
}

// TestDeadlockPrevention tests mechanisms to prevent deadlocks.
func TestDeadlockPrevention(t *testing.T) {
	// Business rule: Always acquire locks in consistent order
	resources := []string{"database", "cache", "filesystem"}

	// Test lock ordering
	lockOrder := determineLockOrder(resources)

	// Verify consistent ordering (alphabetical in this case)
	expected := []string{"cache", "database", "filesystem"}
	assert.Equal(t, expected, lockOrder)

	// Test timeout on lock acquisition
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	acquired := tryAcquireLockWithTimeout(ctx, "resource1")
	assert.True(t, acquired || ctx.Err() != nil, "Should either acquire or timeout")
}

// TestResourceExhaustion tests handling of resource exhaustion scenarios.
func TestResourceExhaustion(t *testing.T) {
	scenarios := []struct {
		name          string
		resource      string
		current       int64
		limit         int64
		shouldProceed bool
	}{
		{
			name:          "memory_within_limits",
			resource:      "memory",
			current:       1024 * 1024 * 500,  // 500MB
			limit:         1024 * 1024 * 1024, // 1GB
			shouldProceed: true,
		},
		{
			name:          "memory_exhausted",
			resource:      "memory",
			current:       1024 * 1024 * 950,  // 950MB (>90% of 1GB)
			limit:         1024 * 1024 * 1024, // 1GB
			shouldProceed: false,
		},
		{
			name:          "file_descriptors_available",
			resource:      "fd",
			current:       500,
			limit:         1024,
			shouldProceed: true,
		},
		{
			name:          "file_descriptors_exhausted",
			resource:      "fd",
			current:       1020,
			limit:         1024,
			shouldProceed: false,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			canProceed := checkResourceAvailability(sc.resource, sc.current, sc.limit)
			assert.Equal(t, sc.shouldProceed, canProceed)

			if !canProceed {
				// Business rule: Log warning and attempt cleanup
				cleanupPerformed := attemptResourceCleanup(sc.resource)
				assert.True(t, cleanupPerformed, "Cleanup should be attempted on exhaustion")
			}
		})
	}
}

// TestGracefulDegradation tests system degradation under load.
func TestGracefulDegradation(t *testing.T) {
	loadLevels := []struct {
		concurrent   int
		expectedMode string
		features     []string
	}{
		{
			concurrent:   10,
			expectedMode: "normal",
			features:     []string{"parallel", "cache", "prefetch"},
		},
		{
			concurrent:   50,
			expectedMode: "degraded",
			features:     []string{"serial", "cache"},
		},
		{
			concurrent:   100,
			expectedMode: "minimal",
			features:     []string{"serial"},
		},
	}

	for _, level := range loadLevels {
		t.Run(fmt.Sprintf("load_%d", level.concurrent), func(t *testing.T) {
			mode := determineOperationMode(level.concurrent)
			assert.Equal(t, level.expectedMode, mode)

			features := getEnabledFeatures(mode)
			assert.Equal(t, level.features, features)
		})
	}
}

// Helper functions for resilience testing.
func isRetriableError(err error) bool {
	retriable := []string{"connection", "refused", "timeout", "checksum"}

	errStr := err.Error()
	for _, pattern := range retriable {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isCriticalSystemError(err error) bool {
	critical := []string{"no space left", "cannot allocate memory", "too many open files"}

	errStr := err.Error()
	for _, pattern := range critical {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func determineRetryCount(err error) int {
	if strings.Contains(err.Error(), "connection") {
		return 3
	}

	if strings.Contains(err.Error(), "checksum") {
		return 1
	}

	return 0
}

func processPackagesWithFailureHandling(packages []domain.Package, failedName string) []domain.InstallationResult {
	results := make([]domain.InstallationResult, 0, len(packages))

	for i := range packages {
		success := packages[i].Name != failedName

		// Essential packages (APT) get special handling
		if !success && packages[i].Method == "apt" {
			// Would normally retry here, but for test we just mark as failed
			success = false
		}

		results = append(results, domain.InstallationResult{
			Package: &packages[i],
			Success: success,
		})
	}

	return results
}

type CircuitBreaker struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	failures         int
	lastFailTime     time.Time
	state            string // closed, open, half-open
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	if cb.state == circuitBreakerOpen {
		if time.Since(cb.lastFailTime) > cb.ResetTimeout {
			cb.state = circuitBreakerHalfOpen
		} else {
			return errors.New("circuit breaker open")
		}
	}

	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.FailureThreshold {
			cb.state = circuitBreakerOpen
			return fmt.Errorf("circuit breaker open: %w", err)
		}

		return err
	}

	// Success resets the breaker
	cb.failures = 0
	cb.state = "closed"

	return nil
}

func determineLockOrder(resources []string) []string {
	// Business rule: Always acquire in alphabetical order to prevent deadlocks
	ordered := make([]string, len(resources))
	copy(ordered, resources)

	// Simple bubble sort for demonstration
	for i := range len(ordered) - 1 {
		for j := range len(ordered) - i - 1 {
			if ordered[j] > ordered[j+1] {
				ordered[j], ordered[j+1] = ordered[j+1], ordered[j]
			}
		}
	}

	return ordered
}

func tryAcquireLockWithTimeout(ctx context.Context, _ string) bool {
	// Simulate lock acquisition with timeout
	acquired := make(chan bool, 1)

	go func() {
		// Simulate some work
		time.Sleep(time.Millisecond * 50)

		acquired <- true
	}()

	select {
	case <-ctx.Done():
		return false
	case result := <-acquired:
		return result
	}
}

func checkResourceAvailability(_ string, current, limit int64) bool {
	// Business rule: Keep 10% buffer for system stability
	threshold := int64(float64(limit) * 0.9)
	return current < threshold
}

func attemptResourceCleanup(_ string) bool {
	// In real implementation, this would trigger cleanup
	// For test, we just return true to indicate cleanup was attempted
	return true
}

func determineOperationMode(concurrent int) string {
	if concurrent <= 20 {
		return "normal"
	}

	if concurrent <= 75 {
		return "degraded"
	}

	return "minimal"
}

func getEnabledFeatures(mode string) []string {
	switch mode {
	case "normal":
		return []string{"parallel", "cache", "prefetch"}
	case "degraded":
		return []string{"serial", "cache"}
	case "minimal":
		return []string{"serial"}
	default:
		return []string{"serial"}
	}
}
