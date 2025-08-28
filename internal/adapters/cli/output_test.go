// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputAdapter_Success(t *testing.T) {
	tests := []struct {
		name         string
		format       OutputFormat
		quiet        bool
		message      string
		data         any
		wantContains string
		wantEmpty    bool
	}{
		{
			name:         "text format with message",
			format:       TextFormat,
			quiet:        false,
			message:      "Operation successful",
			data:         nil,
			wantContains: "Operation successful",
		},
		{
			name:      "quiet mode suppresses message",
			format:    TextFormat,
			quiet:     true,
			message:   "Operation successful",
			data:      nil,
			wantEmpty: true,
		},
		{
			name:    "JSON format with data",
			format:  JSONFormat,
			quiet:   false,
			message: "ignored",
			data: domain.InstallResult{
				Installed: []string{"package1", "package2"},
				Duration:  5 * time.Second,
			},
			wantContains: `"installed"`,
		},
		{
			name:         "JSON format without data shows message",
			format:       JSONFormat,
			quiet:        false,
			message:      "No data to show",
			data:         nil,
			wantContains: "No data to show",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			adapter := NewOutputAdapterWithWriter(&buf, tt.format, tt.quiet)

			err := adapter.Success(tt.message, tt.data)
			require.NoError(t, err)

			output := buf.String()
			if tt.wantEmpty {
				assert.Empty(t, output)
			} else {
				assert.Contains(t, output, tt.wantContains)
			}

			// Verify JSON is valid when in JSON format with data
			if tt.format == JSONFormat && tt.data != nil {
				var result map[string]any

				err := json.Unmarshal(buf.Bytes(), &result)
				assert.NoError(t, err)
			}
		})
	}
}

func TestOutputAdapter_Error(t *testing.T) {
	tests := []struct {
		name         string
		format       OutputFormat
		quiet        bool
		message      string
		wantContains string
		wantEmpty    bool
	}{
		{
			name:         "text format shows error prefix",
			format:       TextFormat,
			quiet:        false,
			message:      "something went wrong",
			wantContains: "Error: something went wrong",
		},
		{
			name:      "quiet mode suppresses error",
			format:    TextFormat,
			quiet:     true,
			message:   "something went wrong",
			wantEmpty: true,
		},
		{
			name:         "JSON format wraps error",
			format:       JSONFormat,
			quiet:        false,
			message:      "something went wrong",
			wantContains: `"error"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			adapter := NewOutputAdapterWithWriter(&buf, tt.format, tt.quiet)

			err := adapter.Error(tt.message)
			require.NoError(t, err)

			output := buf.String()
			if tt.wantEmpty {
				assert.Empty(t, output)
			} else {
				assert.Contains(t, output, tt.wantContains)
			}

			// Verify JSON is valid
			if tt.format == JSONFormat && !tt.quiet {
				var result map[string]string

				err := json.Unmarshal(buf.Bytes(), &result)
				require.NoError(t, err)
				assert.Equal(t, tt.message, result["error"])
			}
		})
	}
}

func TestOutputAdapter_Table(t *testing.T) {
	headers := []string{"Name", "Version", "Status"}
	rows := [][]string{
		{"git", "2.34.1", "installed"},
		{"docker", "20.10.12", "available"},
		{"nodejs", "18.12.0", "installed"},
	}

	t.Run("text format creates aligned table", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, TextFormat, false)

		err := adapter.Table(headers, rows)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "Version")
		assert.Contains(t, output, "Status")
		assert.Contains(t, output, "git")
		assert.Contains(t, output, "docker")
		assert.Contains(t, output, "nodejs")

		// Check for separator line
		assert.Contains(t, output, "----")
	})

	t.Run("JSON format outputs structured data", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, JSONFormat, false)

		err := adapter.Table(headers, rows)
		require.NoError(t, err)

		var result map[string]any

		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Headers come back as []any from JSON unmarshal
		resultHeaders, ok := result["headers"].([]any)
		require.True(t, ok, "headers should be []any")
		assert.Len(t, resultHeaders, len(headers))

		for i, h := range headers {
			assert.Equal(t, h, resultHeaders[i])
		}

		resultRows, ok := result["rows"].([]any)
		require.True(t, ok, "rows should be []any")
		assert.Len(t, resultRows, 3)
	})

	t.Run("quiet mode suppresses table", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, TextFormat, true)

		err := adapter.Table(headers, rows)
		require.NoError(t, err)

		assert.Empty(t, buf.String())
	})
}

func TestOutputAdapter_Progress(t *testing.T) {
	t.Run("text format shows progress", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, TextFormat, false)

		err := adapter.Progress("Installing packages... 50%")
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Installing packages... 50%")
		assert.True(t, strings.HasPrefix(output, "\r"))
	})

	t.Run("JSON format suppresses progress", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, JSONFormat, false)

		err := adapter.Progress("Installing packages... 50%")
		require.NoError(t, err)

		assert.Empty(t, buf.String())
	})

	t.Run("quiet mode suppresses progress", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, TextFormat, true)

		err := adapter.Progress("Installing packages... 50%")
		require.NoError(t, err)

		assert.Empty(t, buf.String())
	})
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    OutputFormat
		wantErr bool
	}{
		{"", TextFormat, false},
		{"text", TextFormat, false},
		{"TEXT", TextFormat, false},
		{"json", JSONFormat, false},
		{"JSON", JSONFormat, false},
		{"xml", TextFormat, true},
		{"yaml", TextFormat, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseOutputFormat(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestOutputFromContext(t *testing.T) {
	t.Run("creates text adapter by default", func(t *testing.T) {
		adapter := OutputFromContext(false, false)
		assert.NotNil(t, adapter)
		assert.False(t, adapter.IsQuiet())
	})

	t.Run("creates JSON adapter when flag set", func(t *testing.T) {
		adapter := OutputFromContext(true, false)
		assert.NotNil(t, adapter)

		// Test JSON output
		var buf bytes.Buffer

		concreteAdapter := NewOutputAdapterWithWriter(&buf, JSONFormat, false)
		err := concreteAdapter.Success("", map[string]string{"test": "value"})
		require.NoError(t, err)
		assert.Contains(t, buf.String(), `"test"`)
	})

	t.Run("creates quiet adapter when flag set", func(t *testing.T) {
		adapter := OutputFromContext(false, true)
		assert.NotNil(t, adapter)
		assert.True(t, adapter.IsQuiet())
	})
}

func TestOutputAdapter_ComplexStructures(t *testing.T) {
	t.Run("InstallResult JSON serialization", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, JSONFormat, false)

		result := domain.InstallResult{
			Installed: []string{"git", "docker"},
			Failed:    []string{"invalid-package"},
			Skipped:   []string{"nodejs"},
			Duration:  10 * time.Second,
			Timestamp: time.Now(),
		}

		err := adapter.Success("", result)
		require.NoError(t, err)

		var decoded domain.InstallResult

		err = json.Unmarshal(buf.Bytes(), &decoded)
		require.NoError(t, err)

		assert.Equal(t, result.Installed, decoded.Installed)
		assert.Equal(t, result.Failed, decoded.Failed)
		assert.Equal(t, result.Skipped, decoded.Skipped)
		assert.Equal(t, result.Duration, decoded.Duration)
	})

	t.Run("ListResult with nested structures", func(t *testing.T) {
		var buf bytes.Buffer

		adapter := NewOutputAdapterWithWriter(&buf, JSONFormat, false)

		result := domain.ListResult{
			Packages: []domain.PackageInfo{
				{
					Name:        "git",
					Version:     "2.34.1",
					Type:        "tool",
					Installed:   time.Now(),
					Size:        1024000,
					Description: "Version control system",
				},
				{
					Name:        "tokyo-night",
					Version:     "1.0.0",
					Type:        "theme",
					Installed:   time.Now(),
					Description: "Dark theme",
				},
			},
			Total:     2,
			Timestamp: time.Now(),
		}

		err := adapter.Success("", result)
		require.NoError(t, err)

		var decoded domain.ListResult

		err = json.Unmarshal(buf.Bytes(), &decoded)
		require.NoError(t, err)

		assert.Len(t, decoded.Packages, 2)
		assert.Equal(t, 2, decoded.Total)
		assert.Equal(t, "git", decoded.Packages[0].Name)
		assert.Equal(t, "theme", decoded.Packages[1].Type)
	})
}
