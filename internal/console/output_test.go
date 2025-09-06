// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package console

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStderr(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w

	f()

	_ = w.Close()
	os.Stderr = old

	out, _ := io.ReadAll(r)

	return string(out)
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)

	return string(out)
}

func TestOutputStateSetMode(t *testing.T) {
	o := &OutputState{}

	o.SetMode(true, false, true)
	assert.True(t, o.Verbose)
	assert.False(t, o.JSON)
	assert.True(t, o.Plain)

	o.SetMode(false, true, false)
	assert.False(t, o.Verbose)
	assert.True(t, o.JSON)
	assert.False(t, o.Plain)
}

func TestOutputStateBold(t *testing.T) {
	tests := []struct {
		name     string
		state    OutputState
		envVars  map[string]string
		input    string
		expected string
	}{
		{
			name:     "plain mode returns unformatted",
			state:    OutputState{Plain: true},
			input:    "test",
			expected: "test",
		},
		{
			name:     "json mode returns unformatted",
			state:    OutputState{JSON: true},
			input:    "test",
			expected: "test",
		},
		{
			name:     "NO_COLOR env disables formatting",
			state:    OutputState{},
			envVars:  map[string]string{"NO_COLOR": "1"},
			input:    "test",
			expected: "test",
		},
		{
			name:     "dumb terminal disables formatting",
			state:    OutputState{},
			envVars:  map[string]string{"TERM": "dumb"},
			input:    "test",
			expected: "test",
		},
		{
			name:     "non-TTY returns uppercase",
			state:    OutputState{},
			input:    "test",
			expected: "TEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result := tt.state.Bold(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOutputStateHeader(t *testing.T) {
	o := &OutputState{}
	// Header just delegates to Bold
	assert.Equal(t, o.Bold("HEADER"), o.Header("header"))
}

func TestOutputStateProgressf(t *testing.T) {
	tests := []struct {
		name         string
		state        OutputState
		expectOutput bool
	}{
		{
			name:         "verbose mode outputs",
			state:        OutputState{Verbose: true},
			expectOutput: true,
		},
		{
			name:         "non-verbose suppresses output",
			state:        OutputState{Verbose: false},
			expectOutput: false,
		},
		{
			name:         "json mode suppresses output",
			state:        OutputState{Verbose: true, JSON: true},
			expectOutput: false,
		},
		{
			name:         "plain mode suppresses output",
			state:        OutputState{Verbose: true, Plain: true},
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(func() {
				tt.state.Progressf("test %s", "message")
			})

			if tt.expectOutput {
				assert.Contains(t, output, "test message")
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestOutputStateSuccessf(t *testing.T) {
	tests := []struct {
		name         string
		state        OutputState
		expectOutput bool
	}{
		{
			name:         "normal mode outputs with checkmark",
			state:        OutputState{},
			expectOutput: true,
		},
		{
			name:         "json mode suppresses output",
			state:        OutputState{JSON: true},
			expectOutput: false,
		},
		{
			name:         "plain mode suppresses output",
			state:        OutputState{Plain: true},
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(func() {
				tt.state.Successf("test %s", "success")
			})

			if tt.expectOutput {
				assert.Contains(t, output, "✓ test success")
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestOutputStateWarningf(t *testing.T) {
	tests := []struct {
		name     string
		state    OutputState
		expected string
	}{
		{
			name:     "normal mode uses warning symbol",
			state:    OutputState{},
			expected: "⚠ test warning",
		},
		{
			name:     "plain mode uses text prefix",
			state:    OutputState{Plain: true},
			expected: "warning: test warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(func() {
				tt.state.Warningf("test %s", "warning")
			})

			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestOutputStateErrorf(t *testing.T) {
	tests := []struct {
		name     string
		state    OutputState
		expected string
	}{
		{
			name:     "normal mode uses error symbol",
			state:    OutputState{},
			expected: "✗ test error",
		},
		{
			name:     "plain mode uses text prefix",
			state:    OutputState{Plain: true},
			expected: "error: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStderr(func() {
				tt.state.Errorf("test %s", "error")
			})

			assert.Contains(t, output, tt.expected)
		})
	}
}

func TestOutputStateResult(t *testing.T) {
	o := &OutputState{}

	output := captureStdout(func() {
		o.Result("test result")
	})

	assert.Equal(t, "test result\n", output)
}

func TestOutputStateJSONResult(t *testing.T) {
	o := &OutputState{}

	output := captureStdout(func() {
		o.JSONResult("success", map[string]any{
			"key": "value",
		})
	})

	var result map[string]any

	err := json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)
	assert.Equal(t, "success", result["status"])
	assert.Equal(t, "value", result["key"])
}

func TestOutputStateSuccessResult(t *testing.T) {
	tests := []struct {
		name         string
		state        OutputState
		result       string
		message      string
		expectJSON   bool
		expectStderr bool
	}{
		{
			name:         "normal mode with message",
			state:        OutputState{},
			result:       "result",
			message:      "success",
			expectJSON:   false,
			expectStderr: true,
		},
		{
			name:         "json mode",
			state:        OutputState{JSON: true},
			result:       "result",
			message:      "success",
			expectJSON:   true,
			expectStderr: false,
		},
		{
			name:         "plain mode",
			state:        OutputState{Plain: true},
			result:       "result",
			message:      "success",
			expectJSON:   false,
			expectStderr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr string

			// Capture both stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			tt.state.SuccessResult(tt.result, tt.message)

			_ = wOut.Close()
			_ = wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			outBytes, _ := io.ReadAll(rOut)
			errBytes, _ := io.ReadAll(rErr)
			stdout = string(outBytes)
			stderr = string(errBytes)

			if tt.expectJSON {
				var result map[string]any

				err := json.Unmarshal([]byte(stdout), &result)
				require.NoError(t, err)
				assert.Equal(t, "success", result["status"])
				assert.Equal(t, tt.result, result["result"])
			} else {
				assert.Contains(t, stdout, tt.result)
			}

			if tt.expectStderr {
				assert.Contains(t, stderr, "✓")
			} else {
				assert.Empty(t, stderr)
			}
		})
	}
}

func TestOutputStateErrorResult(t *testing.T) {
	tests := []struct {
		name       string
		state      OutputState
		err        error
		code       int
		expectJSON bool
	}{
		{
			name:       "normal mode",
			state:      OutputState{},
			err:        errors.New("test error"),
			code:       1,
			expectJSON: false,
		},
		{
			name:       "json mode",
			state:      OutputState{JSON: true},
			err:        errors.New("test error"),
			code:       2,
			expectJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr string

			// Capture both stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			tt.state.ErrorResult(tt.err, tt.code)

			_ = wOut.Close()
			_ = wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			outBytes, _ := io.ReadAll(rOut)
			errBytes, _ := io.ReadAll(rErr)
			stdout = string(outBytes)
			stderr = string(errBytes)

			if tt.expectJSON {
				var result map[string]any

				err := json.Unmarshal([]byte(stdout), &result)
				require.NoError(t, err)
				assert.Equal(t, "error", result["status"])
				assert.Equal(t, "test error", result["error"])
				assert.InEpsilon(t, float64(tt.code), result["code"], 0.01)
			}

			// Error always goes to stderr
			assert.Contains(t, stderr, "test error")
		})
	}
}

func TestOutputStatePlainKeyValue(t *testing.T) {
	o := &OutputState{}

	output := captureStdout(func() {
		o.PlainKeyValue("key", "value")
	})

	assert.Equal(t, "key:value\n", output)
}

func TestOutputStatePlainStatus(t *testing.T) {
	o := &OutputState{}

	output := captureStdout(func() {
		o.PlainStatus("app", "installed")
	})

	assert.Equal(t, "app:installed\n", output)
}

func TestOutputStatePlainList(t *testing.T) {
	o := &OutputState{}

	output := captureStdout(func() {
		o.PlainList([]string{"item1", "item2", "item3"})
	})

	expected := "item1\nitem2\nitem3\n"
	assert.Equal(t, expected, output)
}

func TestOutputStatePlainValue(t *testing.T) {
	o := &OutputState{}

	output := captureStdout(func() {
		o.PlainValue("value")
	})

	assert.Equal(t, "value\n", output)
}

func TestOutputStateIsTTY(t *testing.T) {
	o := &OutputState{}

	// Test with stdout (likely not a TTY in test environment)
	result := o.IsTTY(os.Stdout.Fd())
	assert.False(t, result) // Tests typically run without TTY
}

func TestDefaultOutput(t *testing.T) {
	// Ensure DefaultOutput is initialized
	assert.NotNil(t, DefaultOutput)
	assert.IsType(t, &OutputState{}, DefaultOutput)
}
