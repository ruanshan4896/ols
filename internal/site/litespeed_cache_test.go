package site

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ols-cli/ols/internal/config"
)

func TestGetDefaultLSCachePreset(t *testing.T) {
	preset := GetDefaultLSCachePreset()

	requiredKeys := []string{
		"cache", "cache-mobile", "guest", "guest_optm",
		"instant_click", "optm-browser_cache", "pur-stale",
	}

	for _, k := range requiredKeys {
		if val, exists := preset[k]; !exists || val != 1 {
			t.Errorf("default preset missing or inactive key %s", k)
		}
	}
}

func TestLSCachePresetPathAndSummary(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SystemDir: tmpDir}
	mgr := NewManager(cfg)

	// Ban đầu chưa có file preset
	if mgr.HasLSCachePreset() {
		t.Errorf("expected HasLSCachePreset to be false")
	}

	summary, err := mgr.GetPresetSummary()
	if err != nil {
		t.Fatalf("unexpected error getting summary: %v", err)
	}
	if !strings.Contains(summary, "Chưa có file khuôn mẫu") {
		t.Errorf("expected summary to indicate no preset file, got: %s", summary)
	}

	// Tạo file preset giả lập
	configDir := filepath.Join(tmpDir, "config")
	_ = os.MkdirAll(configDir, 0755)
	sampleConf := map[string]interface{}{
		"cache":         1,
		"cache-mobile":  1,
		"guest":         1,
		"guest_optm":    1,
		"instant_click": 1,
		"optm-css_min":  0,
	}
	sampleData, _ := json.Marshal(sampleConf)
	if err := os.WriteFile(mgr.GetPresetPath(), sampleData, 0644); err != nil {
		t.Fatal(err)
	}

	if !mgr.HasLSCachePreset() {
		t.Errorf("expected HasLSCachePreset to be true")
	}

	summary, err = mgr.GetPresetSummary()
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}

	if !strings.Contains(summary, "Instant Click (Preload)") || !strings.Contains(summary, "BẬT") {
		t.Errorf("expected Instant Click to be BẬT, got summary:\n%s", summary)
	}
	if !strings.Contains(summary, "Minify CSS") || !strings.Contains(summary, "TẮT") {
		t.Errorf("expected Minify CSS to be TẮT, got summary:\n%s", summary)
	}
}

func TestUnzipLSCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Tạo 1 file zip test trong bộ nhớ
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// File 1: plugin main file
	f1, err := zw.Create("litespeed-cache/litespeed-cache.php")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f1.Write([]byte("<?php /* LiteSpeed Cache */"))

	// File 2: file lồng thư mục
	f2, err := zw.Create("litespeed-cache/readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f2.Write([]byte("LiteSpeed Cache Readme"))

	// File 3: Thử nghiệm path traversal độc hại (phải bị bỏ qua)
	f3, err := zw.Create("../hacked.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f3.Write([]byte("malicious"))

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zipFile := filepath.Join(tmpDir, "test.zip")
	if err := os.WriteFile(zipFile, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(tmpDir, "extracted")
	if err := UnzipLSCache(zipFile, destDir); err != nil {
		t.Fatalf("UnzipLSCache failed: %v", err)
	}

	extractedMain := filepath.Join(destDir, "litespeed-cache", "litespeed-cache.php")
	if _, err := os.Stat(extractedMain); err != nil {
		t.Errorf("main plugin file was not extracted: %v", err)
	}

	// Đảm bảo file path traversal không bị giải nén ra ngoài
	hackedPath := filepath.Join(tmpDir, "hacked.txt")
	if _, err := os.Stat(hackedPath); err == nil {
		t.Errorf("security issue: path traversal file was created at %s", hackedPath)
	}
}
