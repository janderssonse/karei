// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package handlers implements CLI command execution logic.
package handlers

import (
	"context"
	"time"

	cliAdapter "github.com/janderssonse/karei/internal/adapters/cli"
	"github.com/janderssonse/karei/internal/domain"
)

// BaseHandler provides common functionality for all command handlers.
type BaseHandler struct {
	Verbose bool
	JSON    bool
	Quiet   bool
	Plain   bool
	Color   string
	Timeout time.Duration
	Output  domain.OutputPort
}

// NewBaseHandler creates a new base handler with the given configuration.
func NewBaseHandler(verbose, json, quiet, plain bool, color string, timeout time.Duration) *BaseHandler {
	return &BaseHandler{
		Verbose: verbose,
		JSON:    json,
		Quiet:   quiet,
		Plain:   plain,
		Color:   color,
		Timeout: timeout,
		Output:  cliAdapter.OutputFromContext(json, quiet),
	}
}

// WithTimeout applies timeout to context if configured.
func (h *BaseHandler) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if h.Timeout > 0 {
		return context.WithTimeout(ctx, h.Timeout)
	}

	return ctx, func() {}
}

// GetOutput returns the output port for CLI rendering.
func (h *BaseHandler) GetOutput() domain.OutputPort {
	if h.Output == nil {
		h.Output = cliAdapter.OutputFromContext(h.JSON, h.Quiet)
	}

	return h.Output
}
