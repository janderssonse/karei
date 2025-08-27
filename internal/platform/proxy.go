// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// GetHTTPClient returns an HTTP client configured with proxy settings.
// Respects HTTP_PROXY, HTTPS_PROXY, and NO_PROXY environment variables.
func GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
}

// GetProxyEnv returns proxy-related environment variables for passing to subprocesses.
// This ensures child processes (apt, curl, wget) inherit proxy settings.
// Returns both uppercase and lowercase versions for maximum compatibility.
//
// Handles essential proxy variables:
//   - http_proxy/HTTP_PROXY: HTTP traffic proxy
//   - https_proxy/HTTPS_PROXY: HTTPS traffic proxy
//   - no_proxy/NO_PROXY: Domains/IPs to bypass proxy (critical for localhost/internal)
func GetProxyEnv() []string {
	var proxyEnv []string

	// Check lowercase first (takes precedence per Unix convention)
	// Export both cases for compatibility with different tools

	// HTTP proxy
	httpProxy := os.Getenv("http_proxy")
	if httpProxy == "" {
		httpProxy = os.Getenv("HTTP_PROXY")
	}

	if httpProxy != "" {
		proxyEnv = append(proxyEnv, "http_proxy="+httpProxy)
		proxyEnv = append(proxyEnv, "HTTP_PROXY="+httpProxy)
	}

	// HTTPS proxy
	httpsProxy := os.Getenv("https_proxy")
	if httpsProxy == "" {
		httpsProxy = os.Getenv("HTTPS_PROXY")
	}

	if httpsProxy != "" {
		proxyEnv = append(proxyEnv, "https_proxy="+httpsProxy)
		proxyEnv = append(proxyEnv, "HTTPS_PROXY="+httpsProxy)
	}

	// NO_PROXY - critical for bypassing proxy for internal services
	noProxy := os.Getenv("no_proxy")
	if noProxy == "" {
		noProxy = os.Getenv("NO_PROXY")
	}

	if noProxy != "" {
		proxyEnv = append(proxyEnv, "no_proxy="+noProxy)
		proxyEnv = append(proxyEnv, "NO_PROXY="+noProxy)
	}

	return proxyEnv
}

// ConfigureAPTProxy returns apt-specific proxy configuration arguments.
// APT requires special -o options for proxy settings.
func ConfigureAPTProxy() []string {
	var args []string

	// Check both cases - lowercase takes precedence per Unix convention
	httpProxy := os.Getenv("http_proxy")
	if httpProxy == "" {
		httpProxy = os.Getenv("HTTP_PROXY")
	}

	if httpProxy != "" {
		args = append(args, "-o", "Acquire::http::Proxy="+httpProxy)
	}

	httpsProxy := os.Getenv("https_proxy")
	if httpsProxy == "" {
		httpsProxy = os.Getenv("HTTPS_PROXY")
	}

	if httpsProxy != "" {
		args = append(args, "-o", "Acquire::https::Proxy="+httpsProxy)
	}

	// Return nil if no proxies configured (not empty slice)
	if len(args) == 0 {
		return nil
	}

	return args
}

// HasProxy checks if any proxy is configured.
// Checks lowercase first (takes precedence per Unix convention).
func HasProxy() bool {
	// Check lowercase first (precedence), then uppercase
	proxyVars := []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY"}
	for _, v := range proxyVars {
		if os.Getenv(v) != "" {
			return true
		}
	}

	return false
}

// IsProxyURL validates if the given string is a valid proxy URL.
func IsProxyURL(proxyStr string) bool {
	if proxyStr == "" {
		return false
	}

	// Handle cases without scheme
	if !strings.HasPrefix(proxyStr, "http://") && !strings.HasPrefix(proxyStr, "https://") && !strings.HasPrefix(proxyStr, "socks5://") {
		proxyStr = "http://" + proxyStr
	}

	_, err := url.Parse(proxyStr)

	return err == nil
}

// GetProxyForURL returns the proxy URL to use for the given target URL.
// Respects NO_PROXY settings.
func GetProxyForURL(targetURL string) string {
	// Let Go's standard library handle the logic
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL, nil)
	if err != nil {
		return ""
	}

	proxyURL, err := http.ProxyFromEnvironment(req)
	if err != nil || proxyURL == nil {
		return ""
	}

	return proxyURL.String()
}
