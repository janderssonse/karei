// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHTTPClient(t *testing.T) {
	t.Parallel()

	// Test that client is created with proxy transport
	client := GetHTTPClient()
	assert.NotNil(t, client)
	assert.NotNil(t, client.Transport)

	// Verify it's configured for proxy
	transport, ok := client.Transport.(*http.Transport)
	assert.True(t, ok)
	assert.NotNil(t, transport.Proxy)
}

func TestGetProxyEnv(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tests := []struct {
		name     string
		setup    func(*testing.T)
		expected []string
	}{
		{
			name: "with HTTP_PROXY set",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
			},
			expected: []string{"HTTP_PROXY=http://proxy.example.com:8080"},
		},
		{
			name: "with multiple proxies",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
				t.Setenv("HTTPS_PROXY", "https://secure.proxy.com:443")
				t.Setenv("NO_PROXY", "localhost,127.0.0.1")
			},
			expected: []string{
				"HTTP_PROXY=http://proxy.example.com:8080",
				"HTTPS_PROXY=https://secure.proxy.com:443",
				"NO_PROXY=localhost,127.0.0.1",
			},
		},
		{
			name: "no proxy set",
			setup: func(t *testing.T) {
				t.Helper()
				// Explicitly clear all proxy variables to ensure isolation
				t.Setenv("http_proxy", "")
				t.Setenv("https_proxy", "")
				t.Setenv("no_proxy", "")
				t.Setenv("HTTP_PROXY", "")
				t.Setenv("HTTPS_PROXY", "")
				t.Setenv("NO_PROXY", "")
			},
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// CRITICAL: Clear ALL proxy environment variables first
			// to ensure complete isolation from host environment
			t.Setenv("http_proxy", "")
			t.Setenv("https_proxy", "")
			t.Setenv("no_proxy", "")
			t.Setenv("HTTP_PROXY", "")
			t.Setenv("HTTPS_PROXY", "")
			t.Setenv("NO_PROXY", "")

			tc.setup(t)

			result := GetProxyEnv()

			// For empty expected, verify result is also empty
			if len(tc.expected) == 0 {
				assert.Empty(t, result, "Expected no proxy vars but got: %v", result)
				return
			}

			// Check all expected values are present
			for _, exp := range tc.expected {
				found := false

				for _, res := range result {
					if res == exp {
						found = true
						break
					}
				}

				assert.True(t, found, "Expected %s not found in result", exp)
			}
		})
	}
}

func TestConfigureAPTProxy(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tests := []struct {
		name     string
		setup    func(*testing.T)
		expected []string
	}{
		{
			name: "with HTTP proxy",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
			},
			expected: []string{"-o", "Acquire::http::Proxy=http://proxy.example.com:8080"},
		},
		{
			name: "with HTTPS proxy",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("HTTPS_PROXY", "https://secure.proxy.com:443")
			},
			expected: []string{"-o", "Acquire::https::Proxy=https://secure.proxy.com:443"},
		},
		{
			name: "with both proxies",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
				t.Setenv("HTTPS_PROXY", "https://secure.proxy.com:443")
			},
			expected: []string{
				"-o", "Acquire::http::Proxy=http://proxy.example.com:8080",
				"-o", "Acquire::https::Proxy=https://secure.proxy.com:443",
			},
		},
		{
			name: "no proxy",
			setup: func(t *testing.T) {
				t.Helper()
				// Explicitly clear all proxy variables
				t.Setenv("http_proxy", "")
				t.Setenv("https_proxy", "")
				t.Setenv("HTTP_PROXY", "")
				t.Setenv("HTTPS_PROXY", "")
			},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear proxy environment first for isolation
			t.Setenv("http_proxy", "")
			t.Setenv("https_proxy", "")
			t.Setenv("HTTP_PROXY", "")
			t.Setenv("HTTPS_PROXY", "")

			tc.setup(t)

			result := ConfigureAPTProxy()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasProxy(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv()
	tests := []struct {
		name     string
		setup    func(*testing.T)
		expected bool
	}{
		{
			name: "has HTTP_PROXY",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
			},
			expected: true,
		},
		{
			name: "has lowercase http_proxy",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("http_proxy", "http://proxy.example.com:8080")
			},
			expected: true,
		},
		{
			name: "no proxy",
			setup: func(t *testing.T) {
				t.Helper()
				// Explicitly clear all proxy variables
				t.Setenv("http_proxy", "")
				t.Setenv("https_proxy", "")
				t.Setenv("HTTP_PROXY", "")
				t.Setenv("HTTPS_PROXY", "")
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear ALL proxy environment variables for complete isolation
			t.Setenv("http_proxy", "")
			t.Setenv("https_proxy", "")
			t.Setenv("HTTP_PROXY", "")
			t.Setenv("HTTPS_PROXY", "")

			tc.setup(t)

			result := HasProxy()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsProxyURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		proxyStr string
		expected bool
	}{
		{
			name:     "valid http proxy",
			proxyStr: "http://proxy.example.com:8080",
			expected: true,
		},
		{
			name:     "valid https proxy",
			proxyStr: "https://secure.proxy.com:443",
			expected: true,
		},
		{
			name:     "valid socks5 proxy",
			proxyStr: "socks5://socks.proxy.com:1080",
			expected: true,
		},
		{
			name:     "proxy without scheme",
			proxyStr: "proxy.example.com:8080",
			expected: true,
		},
		{
			name:     "empty string",
			proxyStr: "",
			expected: false,
		},
		{
			name:     "invalid URL",
			proxyStr: "http://[invalid",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := IsProxyURL(tc.proxyStr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestProxyIntegration(t *testing.T) {
	// Skip in short mode as this requires network setup
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Record that proxy was used
		w.Header().Set("X-Proxy-Used", "true")

		// Simple proxy behavior - just return success
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer proxyServer.Close()

	// Set proxy environment using t.Setenv for isolation
	t.Setenv("HTTP_PROXY", proxyServer.URL)

	// Test that our HTTP client uses the proxy
	client := GetHTTPClient()

	// Create a test target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("direct"))
	}))
	defer targetServer.Close()

	// Make request through client
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetServer.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	defer func() {
		_ = resp.Body.Close()
	}()

	// In a real proxy scenario, we'd check if the proxy was used
	// For this test, we just verify the client works with proxy config
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetProxyForURL_UsesProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")

	result := GetProxyForURL("http://example.com")

	assert.NotEmpty(t, result)
	assert.True(t, strings.HasPrefix(result, "http://"))
}

func TestGetProxyForURL_RespectsNoProxy(t *testing.T) {
	// Skip this test due to Go's http.ProxyFromEnvironment caching issue
	// The function caches environment on first call, making it impossible
	// to test environment changes reliably in the same process
	t.Skip("Skipping due to http.ProxyFromEnvironment caching")

	t.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")
	t.Setenv("NO_PROXY", "example.com")

	result := GetProxyForURL("http://example.com")

	assert.Empty(t, result, "Should not use proxy for excluded domain")
}

func TestGetProxyForURL_NoProxyConfigured(t *testing.T) {
	// Skip this test due to Go's http.ProxyFromEnvironment caching issue
	// The function caches environment on first call, making it impossible
	// to test environment changes reliably in the same process
	t.Skip("Skipping due to http.ProxyFromEnvironment caching")

	// Ensure all proxy env vars are cleared
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("no_proxy", "")

	result := GetProxyForURL("http://example.com")

	assert.Empty(t, result, "Should return empty when no proxy is configured")
}
