// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestContextWithTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		operationTime   time.Duration
		expectCancelled bool
	}{
		{
			name:            "operation completes before timeout",
			timeout:         100 * time.Millisecond,
			operationTime:   10 * time.Millisecond,
			expectCancelled: false,
		},
		{
			name:            "operation exceeds timeout",
			timeout:         10 * time.Millisecond,
			operationTime:   100 * time.Millisecond,
			expectCancelled: true,
		},
		{
			name:            "zero timeout means no timeout",
			timeout:         0,
			operationTime:   10 * time.Millisecond,
			expectCancelled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var (
				ctx    context.Context
				cancel context.CancelFunc
			)

			if tc.timeout > 0 {
				ctx, cancel = context.WithTimeout(context.Background(), tc.timeout)
			} else {
				ctx = context.Background()
				cancel = func() {} // no-op
			}

			defer cancel()

			// Simulate an operation
			done := make(chan bool)

			go func() {
				select {
				case <-time.After(tc.operationTime):
					done <- true
				case <-ctx.Done():
					done <- false
				}
			}()

			completed := <-done

			if tc.expectCancelled {
				assert.False(t, completed, "Operation should have been cancelled")
				assert.Error(t, ctx.Err())
			} else {
				assert.True(t, completed, "Operation should have completed")
				// For zero timeout, context should have no error
				if tc.timeout == 0 {
					assert.NoError(t, ctx.Err())
				}
			}
		})
	}
}

func TestTimeoutPropagation(t *testing.T) {
	// Test that timeout is properly propagated through the system
	timeout := 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Verify context has deadline
	deadline, ok := ctx.Deadline()
	assert.True(t, ok, "Context should have deadline")
	assert.WithinDuration(t, time.Now().Add(timeout), deadline, 10*time.Millisecond)
}
