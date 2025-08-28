// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandUtils_CommandExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{
			name:    "existing command",
			command: "ls",
			want:    true,
		},
		{
			name:    "non-existing command",
			command: "nonexistentcommand12345",
			want:    false,
		},
		{
			name:    "empty command",
			command: "",
			want:    false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := CommandExists(testCase.command)
			require.Equal(t, testCase.want, got)
		})
	}
}
