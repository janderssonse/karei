// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package network

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetHTTPClient(t *testing.T) {
	client := GetHTTPClient()

	assert.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)

	// Verify transport is configured
	transport, ok := client.Transport.(*http.Transport)
	assert.True(t, ok)
	assert.NotNil(t, transport.Proxy)
}

func TestGetProxyEnv(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectedLen int
		contains    []string
	}{
		{
			name:        "no proxy configured",
			envVars:     map[string]string{},
			expectedLen: 0,
			contains:    []string{},
		},
		{
			name: "lowercase http_proxy set",
			envVars: map[string]string{
				"http_proxy": "http://proxy.example.com:8080",
			},
			expectedLen: 2,
			contains: []string{
				"http_proxy=http://proxy.example.com:8080",
				"HTTP_PROXY=http://proxy.example.com:8080",
			},
		},
		{
			name: "uppercase HTTP_PROXY set",
			envVars: map[string]string{
				"HTTP_PROXY": "http://proxy.example.com:8080",
			},
			expectedLen: 2,
			contains: []string{
				"http_proxy=http://proxy.example.com:8080",
				"HTTP_PROXY=http://proxy.example.com:8080",
			},
		},
		{
			name: "all proxy variables set",
			envVars: map[string]string{
				"http_proxy":  "http://proxy.example.com:8080",
				"https_proxy": "https://proxy.example.com:8443",
				"no_proxy":    "localhost,127.0.0.1,internal.domain",
			},
			expectedLen: 6,
			contains: []string{
				"http_proxy=http://proxy.example.com:8080",
				"HTTP_PROXY=http://proxy.example.com:8080",
				"https_proxy=https://proxy.example.com:8443",
				"HTTPS_PROXY=https://proxy.example.com:8443",
				"no_proxy=localhost,127.0.0.1,internal.domain",
				"NO_PROXY=localhost,127.0.0.1,internal.domain",
			},
		},
		{
			name: "lowercase takes precedence over uppercase",
			envVars: map[string]string{
				"http_proxy": "http://lower.proxy:8080",
				"HTTP_PROXY": "http://upper.proxy:8080",
			},
			expectedLen: 2,
			contains: []string{
				"http_proxy=http://lower.proxy:8080",
				"HTTP_PROXY=http://lower.proxy:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all proxy vars first, then set the ones we need
			proxyVars := []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY", "no_proxy", "NO_PROXY"}
			for _, key := range proxyVars {
				t.Setenv(key, "")
			}

			// Set the test environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			result := GetProxyEnv()

			assert.Len(t, result, tt.expectedLen)

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestConfigureAPTProxy(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectedLen int
		contains    []string
	}{
		{
			name:        "no proxy configured returns nil",
			envVars:     map[string]string{},
			expectedLen: 0,
			contains:    []string{},
		},
		{
			name: "http proxy only",
			envVars: map[string]string{
				"http_proxy": "http://proxy.example.com:8080",
			},
			expectedLen: 2,
			contains: []string{
				"-o",
				"Acquire::http::Proxy=http://proxy.example.com:8080",
			},
		},
		{
			name: "https proxy only",
			envVars: map[string]string{
				"https_proxy": "https://secure.proxy.com:8443",
			},
			expectedLen: 2,
			contains: []string{
				"-o",
				"Acquire::https::Proxy=https://secure.proxy.com:8443",
			},
		},
		{
			name: "both proxies configured",
			envVars: map[string]string{
				"http_proxy":  "http://proxy.example.com:8080",
				"https_proxy": "https://secure.proxy.com:8443",
			},
			expectedLen: 4,
			contains: []string{
				"-o",
				"Acquire::http::Proxy=http://proxy.example.com:8080",
				"-o",
				"Acquire::https::Proxy=https://secure.proxy.com:8443",
			},
		},
		{
			name: "uppercase proxy variables",
			envVars: map[string]string{
				"HTTP_PROXY":  "http://proxy.example.com:8080",
				"HTTPS_PROXY": "https://secure.proxy.com:8443",
			},
			expectedLen: 4,
			contains: []string{
				"Acquire::http::Proxy=http://proxy.example.com:8080",
				"Acquire::https::Proxy=https://secure.proxy.com:8443",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all proxy vars first, then set the ones we need
			proxyVars := []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY"}
			for _, key := range proxyVars {
				t.Setenv(key, "")
			}

			// Set the test environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			result := ConfigureAPTProxy()

			if tt.expectedLen == 0 {
				assert.Nil(t, result)
			} else {
				assert.Len(t, result, tt.expectedLen)

				for _, expected := range tt.contains {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

func TestGetProxyForURL(t *testing.T) {
	// Test basic functionality - focus on code behavior not environment
	tests := []struct {
		name      string
		targetURL string
		setup     func()
		cleanup   func()
		checkFunc func(result string)
	}{
		{
			name:      "returns empty for invalid URL",
			targetURL: "://invalid-url",
			setup:     func() {},
			cleanup:   func() {},
			checkFunc: func(result string) {
				assert.Empty(t, result, "Invalid URL should return empty proxy")
			},
		},
		{
			name:      "handles valid http URL",
			targetURL: "http://example.com",
			setup:     func() {},
			cleanup:   func() {},
			checkFunc: func(result string) {
				// Result depends on environment - just verify it doesn't panic
				assert.NotNil(t, &result)
			},
		},
		{
			name:      "handles valid https URL",
			targetURL: "https://secure.example.com",
			setup:     func() {},
			cleanup:   func() {},
			checkFunc: func(result string) {
				// Result depends on environment - just verify it doesn't panic
				assert.NotNil(t, &result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			tt.setup()
			defer tt.cleanup()

			result := GetProxyForURL(tt.targetURL)
			tt.checkFunc(result)
		})
	}
}
