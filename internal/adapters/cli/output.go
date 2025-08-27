// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package cli provides output adapters for CLI operations.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/janderssonse/karei/internal/domain"
)

var (
	// ErrUnsupportedFormat is returned when an unsupported output format is requested.
	ErrUnsupportedFormat = errors.New("unsupported output format")
)

// OutputAdapter implements domain.OutputPort for CLI output.
type OutputAdapter struct {
	writer io.Writer
	format OutputFormat
	quiet  bool
}

// OutputFormat represents the output format type.
type OutputFormat int

const (
	// TextFormat outputs human-readable text.
	TextFormat OutputFormat = iota
	// JSONFormat outputs machine-readable JSON.
	JSONFormat
)

// NewOutputAdapter creates a new output adapter with the specified configuration.
func NewOutputAdapter(format OutputFormat, quiet bool) *OutputAdapter {
	return &OutputAdapter{
		writer: os.Stdout,
		format: format,
		quiet:  quiet,
	}
}

// NewOutputAdapterWithWriter creates a new output adapter with a custom writer for testing.
func NewOutputAdapterWithWriter(writer io.Writer, format OutputFormat, quiet bool) *OutputAdapter {
	return &OutputAdapter{
		writer: writer,
		format: format,
		quiet:  quiet,
	}
}

// Success outputs a success message with optional structured data.
func (o *OutputAdapter) Success(message string, data interface{}) error {
	if o.quiet && data == nil {
		return nil
	}

	if o.format == JSONFormat && data != nil {
		return o.outputJSON(data)
	}

	if message != "" && !o.quiet {
		_, _ = fmt.Fprintln(o.writer, message)
	}

	return nil
}

// Error outputs an error message.
func (o *OutputAdapter) Error(message string) error {
	if o.quiet {
		return nil
	}

	if o.format == JSONFormat {
		errorData := map[string]string{"error": message}

		return o.outputJSON(errorData)
	}

	_, _ = fmt.Fprintf(o.writer, "Error: %s\n", message)

	return nil
}

// Info outputs an informational message.
func (o *OutputAdapter) Info(message string) error {
	if o.quiet {
		return nil
	}

	if o.format == JSONFormat {
		infoData := map[string]string{"info": message}

		return o.outputJSON(infoData)
	}

	_, _ = fmt.Fprintln(o.writer, message)

	return nil
}

// Progress outputs progress information for long-running operations.
func (o *OutputAdapter) Progress(message string) error {
	if o.quiet || o.format == JSONFormat {
		return nil
	}

	_, _ = fmt.Fprintf(o.writer, "\r%s", message)

	return nil
}

// Table outputs tabular data.
func (o *OutputAdapter) Table(headers []string, rows [][]string) error {
	if o.quiet {
		return nil
	}

	if o.format == JSONFormat {
		tableData := map[string]interface{}{
			"headers": headers,
			"rows":    rows,
		}

		return o.outputJSON(tableData)
	}

	w := tabwriter.NewWriter(o.writer, 0, 0, 2, ' ', 0)

	defer func() { _ = w.Flush() }()

	// Print headers
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))

	// Print separator
	separators := make([]string, len(headers))
	for i := range headers {
		separators[i] = strings.Repeat("-", len(headers[i]))
	}

	_, _ = fmt.Fprintln(w, strings.Join(separators, "\t"))

	// Print rows
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	return nil
}

// IsQuiet returns true if output should be suppressed.
func (o *OutputAdapter) IsQuiet() bool {
	return o.quiet
}

// outputJSON outputs data as JSON.
func (o *OutputAdapter) outputJSON(data interface{}) error {
	encoder := json.NewEncoder(o.writer)
	encoder.SetIndent("", "  ")

	return encoder.Encode(data)
}

// ParseOutputFormat parses a string into an OutputFormat.
func ParseOutputFormat(format string) (OutputFormat, error) {
	switch strings.ToLower(format) {
	case "", "text":
		return TextFormat, nil
	case "json":
		return JSONFormat, nil
	default:
		return TextFormat, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
}

// OutputFromContext creates an OutputAdapter from CLI context flags.
func OutputFromContext(jsonFlag, quietFlag bool) domain.OutputPort {
	format := TextFormat
	if jsonFlag {
		format = JSONFormat
	}

	return NewOutputAdapter(format, quietFlag)
}
