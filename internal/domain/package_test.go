// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package domain

import (
	"testing"
)

func TestPackage_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pkg      *Package
		expected bool
	}{
		{
			name: "valid package",
			pkg: &Package{
				Name:   "vim",
				Method: MethodAPT,
				Source: "vim",
			},
			expected: true,
		},
		{
			name: "missing name",
			pkg: &Package{
				Method: MethodAPT,
				Source: "vim",
			},
			expected: false,
		},
		{
			name: "missing method",
			pkg: &Package{
				Name:   "vim",
				Source: "vim",
			},
			expected: false,
		},
		{
			name: "missing source",
			pkg: &Package{
				Name:   "vim",
				Method: MethodAPT,
			},
			expected: false,
		},
		{
			name:     "nil package",
			pkg:      nil,
			expected: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var result bool
			if testCase.pkg != nil {
				result = testCase.pkg.IsValid()
			}

			if result != testCase.expected {
				t.Errorf("Package.IsValid() = %v, expected %v", result, testCase.expected)
			}
		})
	}
}
