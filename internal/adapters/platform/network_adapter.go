// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/janderssonse/karei/internal/adapters/network"
	"github.com/janderssonse/karei/internal/domain"
)

// NetworkAdapter provides network operations.
type NetworkAdapter struct{}

// NewNetworkAdapter creates a new network adapter.
func NewNetworkAdapter() domain.NetworkClient {
	return &NetworkAdapter{}
}

// DownloadFile downloads a file from a URL to a destination path.
func (n *NetworkAdapter) DownloadFile(ctx context.Context, url, destPath string) error {
	client := network.GetHTTPClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create the destination file
	out, err := os.Create(filepath.Clean(destPath))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	defer func() {
		_ = out.Close()
	}()

	// Copy the response body to the file
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
