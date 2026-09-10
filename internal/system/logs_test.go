package system

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseTopIPs(t *testing.T) {
	raw := `192.168.1.100 - - [10/Sep/2026:15:30:09 +0700] "POST /xmlrpc.php HTTP/1.1" 200 36
192.168.1.100 - - [10/Sep/2026:15:30:10 +0700] "GET / HTTP/1.1" 200 17674
10.0.0.5 - - [10/Sep/2026:15:30:15 +0700] "GET /the-thao/ HTTP/1.1" 200 17711
192.168.1.100 - - [10/Sep/2026:15:30:17 +0700] "POST /xmlrpc.php HTTP/1.1" 200 36
10.0.0.5 - - [10/Sep/2026:15:30:19 +0700] "GET /casino/ HTTP/1.1" 200 17851
1.2.3.4 - - [10/Sep/2026:15:30:21 +0700] "GET /tin-tuc/ HTTP/1.1" 200 17629
`
	stats := ParseTopIPs(raw, 5)
	if len(stats) != 3 {
		t.Fatalf("expected 3 distinct IPs, got %d", len(stats))
	}

	// 192.168.1.100 xuất hiện 3 lần
	if stats[0].IP != "192.168.1.100" || stats[0].Count != 3 {
		t.Errorf("expected top IP 192.168.1.100 with count 3, got %+v", stats[0])
	}
	// 10.0.0.5 xuất hiện 2 lần
	if stats[1].IP != "10.0.0.5" || stats[1].Count != 2 {
		t.Errorf("expected second IP 10.0.0.5 with count 2, got %+v", stats[1])
	}
}

func TestParseTopURLs(t *testing.T) {
	raw := `192.168.1.1 - - [10/Sep/2026:15:30:09] "POST /xmlrpc.php HTTP/1.1" 200 36
192.168.1.2 - - [10/Sep/2026:15:30:10] "POST /xmlrpc.php HTTP/1.1" 200 36
192.168.1.3 - - [10/Sep/2026:15:30:15] "GET /the-thao/ HTTP/1.1" 200 17711
192.168.1.4 - - [10/Sep/2026:15:30:17] "POST /xmlrpc.php HTTP/1.1" 200 36
192.168.1.5 - - [10/Sep/2026:15:30:19] "GET / HTTP/1.1" 200 17851
`
	stats := ParseTopURLs(raw, 5)
	if len(stats) != 3 {
		t.Fatalf("expected 3 distinct URLs, got %d", len(stats))
	}

	if stats[0].URL != "/xmlrpc.php" || stats[0].Count != 3 {
		t.Errorf("expected top URL /xmlrpc.php with count 3, got %+v", stats[0])
	}
}

func TestGetSiteAccessAndErrorLog_HostFile(t *testing.T) {
	tmpDir := t.TempDir()
	domain := "example.com"
	logsDir := filepath.Join(tmpDir, "sites", domain, "logs")
	_ = os.MkdirAll(logsDir, 0755)

	accessFile := filepath.Join(logsDir, "access.log")
	errorFile := filepath.Join(logsDir, "error.log")

	_ = os.WriteFile(accessFile, []byte("1.1.1.1 - [access log 1]\n2.2.2.2 - [access log 2]\n"), 0644)
	_ = os.WriteFile(errorFile, []byte("PHP Fatal error: something crashed\n"), 0644)

	// Đọc access log
	accessContent, err := GetSiteAccessLog(tmpDir, domain, 10)
	if err != nil {
		t.Fatalf("unexpected error reading access log: %v", err)
	}
	if !strings.Contains(accessContent, "1.1.1.1") || !strings.Contains(accessContent, "2.2.2.2") {
		t.Errorf("expected access log content, got: %s", accessContent)
	}

	// Đọc error log
	errContent, err := GetSiteErrorLog(tmpDir, domain, 10)
	if err != nil {
		t.Fatalf("unexpected error reading error log: %v", err)
	}
	if !strings.Contains(errContent, "Fatal error") {
		t.Errorf("expected error log content, got: %s", errContent)
	}
}
