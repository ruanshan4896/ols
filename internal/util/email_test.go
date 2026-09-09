package util

import (
	"testing"
)

func TestValidateAndSanitizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		hasErr   bool
	}{
		{"ruanshan4896@gmail.com", "ruanshan4896@gmail.com", false},
		{"RuAnShan4896@Gmail.Com", "ruanshan4896@gmail.com", false},
		// Test có dấu tiếng Việt do Unikey
		{"ruấnshan4896@gmail.com", "ruanshan4896@gmail.com", false},
		{" admin@domain.com  ", "admin@domain.com", false},
		{"", "", true},
		{"invalid-email", "", true},
	}

	for _, tt := range tests {
		actual, err := ValidateAndSanitizeEmail(tt.input)
		if (err != nil) != tt.hasErr {
			t.Errorf("ValidateAndSanitizeEmail(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if !tt.hasErr && actual != tt.expected {
			t.Errorf("ValidateAndSanitizeEmail(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}
