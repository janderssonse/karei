package main

import (
	"fmt"
	"github.com/janderssonse/karei/internal/network"
	"os"
)

func main() {
	// Clear all
	proxyVars := []string{"http_proxy", "https_proxy", "no_proxy", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}
	for _, key := range proxyVars {
		_ = os.Unsetenv(key)
	}

	// Set http_proxy
	_ = os.Setenv("http_proxy", "http://proxy.example.com:8080")

	result := network.GetProxyForURL("http://example.com")
	fmt.Printf("Result: '%s'\n", result)
}
