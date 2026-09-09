package system

import (
	"testing"
)

func TestFilterErrorLines(t *testing.T) {
	raw := `[INFO] Server started on port 80
[ERROR] Failed to obtain SSL certificate for domain: acme: error 400
[DEBUG] Routing rule matched
[FATAL] Database connection refused: host 127.0.0.1:3306
[INFO] Request served in 45ms
[WARN] High memory usage detected
PHP Fatal error: Uncaught Error: Call to undefined function in wp-content/plugins/test.php
`

	entries := FilterErrorLines("Traefik", raw)

	if len(entries) != 3 {
		t.Fatalf("expected 3 error entries, got %d", len(entries))
	}

	if entries[0].Source != "Traefik" {
		t.Errorf("expected source Traefik, got %s", entries[0].Source)
	}
}

func TestFilterErrorLinesEmpty(t *testing.T) {
	raw := `[INFO] Everything is running smoothly
[DEBUG] No problems found
`
	entries := FilterErrorLines("MariaDB", raw)
	if len(entries) != 0 {
		t.Errorf("expected 0 error entries, got %d", len(entries))
	}
}
