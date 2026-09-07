package config

import (
	"path/filepath"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "ols.yaml")

	cfg := DefaultConfig()
	cfg.ACMEEmail = "admin@example.com"
	cfg.DBRootPassword = "secretpassword123"

	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.ACMEEmail != "admin@example.com" {
		t.Errorf("expected email admin@example.com, got %s", loaded.ACMEEmail)
	}
	if loaded.DBRootPassword != "secretpassword123" {
		t.Errorf("expected password secretpassword123, got %s", loaded.DBRootPassword)
	}
	if loaded.SystemDir != "/opt/ols" {
		t.Errorf("expected system dir /opt/ols, got %s", loaded.SystemDir)
	}
}
