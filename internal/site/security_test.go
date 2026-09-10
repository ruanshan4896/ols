package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ols-cli/ols/internal/config"
)

func TestFetchWordPressSalts(t *testing.T) {
	salts, err := FetchWordPressSalts()
	if err != nil {
		t.Fatalf("FetchWordPressSalts failed: %v", err)
	}

	if !strings.Contains(salts, "AUTH_KEY") || !strings.Contains(salts, "NONCE_SALT") {
		t.Errorf("Returned salts missing expected keys: %s", salts)
	}
}

func TestRegenerateSalts(t *testing.T) {
	tmpDir := t.TempDir()
	domain := "test-sec.com"
	siteDir := filepath.Join(tmpDir, "sites", domain)
	htmlDir := filepath.Join(siteDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	initialConfig := `<?php
define( 'DB_NAME', 'wp_test' );
// Authentication Unique Keys and Salts
define( 'AUTH_KEY',         'old-key-1' );
define( 'SECURE_AUTH_KEY',  'old-key-2' );
define( 'LOGGED_IN_KEY',    'old-key-3' );
define( 'NONCE_KEY',        'old-key-4' );
define( 'AUTH_SALT',        'old-key-5' );
define( 'SECURE_AUTH_SALT', 'old-key-6' );
define( 'LOGGED_IN_SALT',   'old-key-7' );
define( 'NONCE_SALT',       'old-key-8' );

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}
`
	wpConfigPath := filepath.Join(htmlDir, "wp-config.php")
	if err := os.WriteFile(wpConfigPath, []byte(initialConfig), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := &Manager{
		cfg: &config.Config{
			SystemDir: tmpDir,
		},
	}

	ok, err := mgr.RegenerateSalts(domain)
	if err != nil || !ok {
		t.Fatalf("RegenerateSalts failed: %v", err)
	}

	updatedBytes, err := os.ReadFile(wpConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	updatedContent := string(updatedBytes)
	if strings.Contains(updatedContent, "old-key-1") {
		t.Errorf("Old salts were not replaced properly: %s", updatedContent)
	}
	if !strings.Contains(updatedContent, "AUTH_KEY") || !strings.Contains(updatedContent, "NONCE_SALT") {
		t.Errorf("New salts missing in wp-config.php: %s", updatedContent)
	}
}

func TestGetSitePHPBinary(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SystemDir: tmpDir}
	mgr := NewManager(cfg)

	// Case 1: Site with PHP 8.3 compose
	siteDir83 := filepath.Join(tmpDir, "sites", "site83.com")
	_ = os.MkdirAll(siteDir83, 0755)
	_ = os.WriteFile(filepath.Join(siteDir83, "docker-compose.yml"), []byte("services:\n  ols:\n    image: litespeedtech/openlitespeed:1.8.2-lsphp83\n"), 0644)
	if bin := mgr.GetSitePHPBinary("site83.com"); bin != "/usr/local/lsws/lsphp83/bin/php" {
		t.Errorf("expected /usr/local/lsws/lsphp83/bin/php, got %s", bin)
	}

	// Case 2: Default fallback to 8.2
	if bin := mgr.GetSitePHPBinary("nonexistent.com"); bin != "/usr/local/lsws/lsphp82/bin/php" {
		t.Errorf("expected default /usr/local/lsws/lsphp82/bin/php, got %s", bin)
	}
}

func TestReplaceSaltsInWPConfig_DollarSignAndSpecialChars(t *testing.T) {
	initial := `<?php
define( 'DB_NAME', 'wp_test' );
// Authentication Unique Keys and Salts
define( 'AUTH_KEY',         'old1' );
define( 'SECURE_AUTH_KEY',  'old2' );
define( 'LOGGED_IN_KEY',    'old3' );
define( 'NONCE_KEY',        'old4' );
define( 'AUTH_SALT',        'old5' );
define( 'SECURE_AUTH_SALT', 'old6' );
define( 'LOGGED_IN_SALT',   'old7' );
define( 'NONCE_SALT',       'old8' );

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}
`

	// Salts containing $0, $1, $2, ${foo}, $`, etc. which would break ReplaceAllString
	complexSalts := `define('AUTH_KEY',         'a$0b$1c$2d$name');
define('SECURE_AUTH_KEY',  'test${foo}bar');
define('LOGGED_IN_KEY',    'val$$double');
define('NONCE_KEY',        'key$');
define('AUTH_SALT',        'salt$7end');
define('SECURE_AUTH_SALT', 'salt}curly');
define('LOGGED_IN_SALT',   'salt{sk<#%');
define('NONCE_SALT',       'final$1$2salt');`

	result := ReplaceSaltsInWPConfig(initial, complexSalts)

	// Verify all special sequences are preserved verbatim
	expectedSubstrings := []string{
		"a$0b$1c$2d$name",
		"test${foo}bar",
		"val$$double",
		"salt$7end",
		"salt}curly",
		"salt{sk<#%",
		"final$1$2salt",
		"if ( ! defined( 'ABSPATH' ) ) {",
	}

	for _, exp := range expectedSubstrings {
		if !strings.Contains(result, exp) {
			t.Errorf("expected result to contain %q, but got:\n%s", exp, result)
		}
	}

	// Verify old salts are completely removed
	if strings.Contains(result, "old1") || strings.Contains(result, "old8") {
		t.Errorf("old salts still present in result:\n%s", result)
	}

	// Test regenerating a second time (with previously generated comment)
	secondSalts := `define('AUTH_KEY',         'second-round-key');
define('NONCE_SALT',       'second-round-nonce');`
	secondResult := ReplaceSaltsInWPConfig(result, secondSalts)

	if !strings.Contains(secondResult, "second-round-key") {
		t.Errorf("second regeneration failed to replace salts")
	}
	if strings.Contains(secondResult, "a$0b$1c$2d$name") {
		t.Errorf("first round salts still present after second round")
	}
}

