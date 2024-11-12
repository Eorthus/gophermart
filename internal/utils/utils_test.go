package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "basic split",
			input:    "a:b:c",
			sep:      ":",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "no separator",
			input:    "abc",
			sep:      ":",
			expected: []string{"abc"},
		},
		{
			name:     "empty string",
			input:    "",
			sep:      ":",
			expected: nil,
		},
		{
			name:     "multiple character separator",
			input:    "a::b::c",
			sep:      "::",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "separator at start",
			input:    ":a:b:c",
			sep:      ":",
			expected: []string{"", "a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitString(tt.input, tt.sep)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		name     string
		number   string
		expected bool
	}{
		{
			name:     "valid number 1",
			number:   "4532015112830366",
			expected: true,
		},
		{
			name:     "invalid number",
			number:   "4532015112830367",
			expected: false,
		},
		{
			name:     "empty string",
			number:   "",
			expected: false,
		},
		{
			name:     "non-numeric string",
			number:   "12345a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateLuhn(tt.number)
			assert.Equal(t, tt.expected, result)
		})
	}
}
