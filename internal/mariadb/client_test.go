package mariadb

import "testing"

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"wp_site_a", "wp_site_a"},
		{"wp-site.a", "wp_site_a"},
		{"drop;database", "drop_database"},
	}

	for _, tt := range tests {
		actual := SanitizeIdentifier(tt.input)
		if actual != tt.expected {
			t.Errorf("SanitizeIdentifier(%q) = %q; want %q", tt.input, actual, tt.expected)
		}
	}
}
