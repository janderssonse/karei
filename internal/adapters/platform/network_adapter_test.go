// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package platform_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkAdapter_DownloadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serverFunc  func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		wantContent string
	}{
		{
			name: "successful download",
			serverFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("test file content"))
			},
			wantErr:     false,
			wantContent: "test file content",
		},
		{
			name: "server returns 404",
			serverFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
		{
			name: "server returns 500",
			serverFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "large file download",
			serverFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				// Write 1MB of data
				data := make([]byte, 1024*1024)
				for i := range data {
					data[i] = byte(i % 256)
				}
				_, _ = w.Write(data)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverFunc))
			defer server.Close()

			// Create temp file
			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "download.txt")

			// Create network adapter
			adapter := platform.NewNetworkAdapter()

			// Download file
			err := adapter.DownloadFile(context.Background(), server.URL, destPath)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify file exists and has content
				content, err := os.ReadFile(filepath.Clean(destPath))
				require.NoError(t, err)

				if tt.wantContent != "" {
					assert.Equal(t, tt.wantContent, string(content))
				}
			}
		})
	}
}

func TestNetworkAdapter_DownloadFileWithContext(t *testing.T) {
	t.Parallel()

	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate slow download
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slow content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "download.txt")

	adapter := platform.NewNetworkAdapter()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Download should fail due to timeout
	err := adapter.DownloadFile(ctx, server.URL, destPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestNetworkAdapter_InvalidURL(t *testing.T) {
	t.Parallel()

	adapter := platform.NewNetworkAdapter()
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "download.txt")

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "invalid URL scheme",
			url:     "ftp://example.com/file",
			wantErr: true,
		},
		{
			name:    "malformed URL",
			url:     "://not-a-url",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "unreachable host",
			url:     "http://nonexistent.invalid.domain.test/file",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := adapter.DownloadFile(context.Background(), tt.url, destPath)
			if tt.wantErr {
				require.Error(t, err)
			}
		})
	}
}

func TestNetworkAdapter_WritePermissions(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	}))
	defer server.Close()

	// Try to write to a read-only directory
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(readOnlyDir, 0750))

	destPath := filepath.Join(readOnlyDir, "file.txt")

	// Make directory read-only
	require.NoError(t, os.Chmod(readOnlyDir, 0400))

	defer func() {
		_ = os.Chmod(readOnlyDir, 0600) // Restore permissions for cleanup
	}()

	adapter := platform.NewNetworkAdapter()
	err := adapter.DownloadFile(context.Background(), server.URL, destPath)

	// Should fail due to permissions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission")
}
