// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package stringutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testSentence = "The quick brown fox jumps over the lazy dog"

func TestContainsAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		text       string
		substrings []string
		want       bool
	}{
		{
			name:       "single match",
			text:       "hello world",
			substrings: []string{"world"},
			want:       true,
		},
		{
			name:       "multiple substrings with match",
			text:       "hello world",
			substrings: []string{"foo", "world", "bar"},
			want:       true,
		},
		{
			name:       "no match",
			text:       "hello world",
			substrings: []string{"foo", "bar", "baz"},
			want:       false,
		},
		{
			name:       "empty substrings list",
			text:       "hello world",
			substrings: []string{},
			want:       false,
		},
		{
			name:       "empty text",
			text:       "",
			substrings: []string{"foo"},
			want:       false,
		},
		{
			name:       "empty string in substrings",
			text:       "hello world",
			substrings: []string{""},
			want:       true, // Every string contains empty string
		},
		{
			name:       "case sensitive match",
			text:       "Hello World",
			substrings: []string{"hello", "world"},
			want:       false,
		},
		{
			name:       "partial match",
			text:       "hello world",
			substrings: []string{"ello", "orl"},
			want:       true,
		},
		{
			name:       "special characters",
			text:       "hello@world.com",
			substrings: []string{"@", ".", "com"},
			want:       true,
		},
		{
			name:       "unicode characters",
			text:       "hello 世界",
			substrings: []string{"世界", "你好"},
			want:       true,
		},
		{
			name:       "newlines and tabs",
			text:       "hello\nworld\ttab",
			substrings: []string{"\n", "\t"},
			want:       true,
		},
		{
			name:       "nil substrings",
			text:       "hello world",
			substrings: nil,
			want:       false,
		},
		{
			name:       "all substrings match",
			text:       "hello world",
			substrings: []string{"hello", "world", "ello", "orld"},
			want:       true,
		},
		{
			name:       "long text with match at end",
			text:       "this is a very long text with many words and the match is at the end",
			substrings: []string{"beginning", "middle", "end"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ContainsAny(tt.text, tt.substrings)
			assert.Equal(t, tt.want, got)
		})
	}
}

func BenchmarkContainsAny(b *testing.B) {
	text := testSentence
	substrings := []string{"cat", "dog", "bird", "fish"}

	b.ResetTimer()

	for range b.N {
		_ = ContainsAny(text, substrings)
	}
}

func BenchmarkContainsAnyNoMatch(b *testing.B) {
	text := testSentence
	substrings := []string{"cat", "bird", "fish", "elephant"}

	b.ResetTimer()

	for range b.N {
		_ = ContainsAny(text, substrings)
	}
}

func BenchmarkContainsAnyLongList(b *testing.B) {
	text := testSentence

	substrings := make([]string, 100)
	for i := range substrings {
		substrings[i] = fmt.Sprintf("word%d", i)
	}

	substrings[99] = "dog" // Match at the end

	b.ResetTimer()

	for range b.N {
		_ = ContainsAny(text, substrings)
	}
}
