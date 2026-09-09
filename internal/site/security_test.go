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
