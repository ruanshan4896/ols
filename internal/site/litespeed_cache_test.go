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
		"instant_click", "optm-browser_cache", "purge-stale",
	}

	for _, k := range requiredKeys {
		val, exists := preset["litespeed."+k]
		if !exists {
			val, exists = preset[k]
		}
		if !exists || val != 1 {
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

func TestParseApplyOutput(t *testing.T) {
	// Case 1: Chuẩn SUCCESS đơn thuần
	if err := ParseApplyOutput("SUCCESS"); err != nil {
		t.Errorf("expected nil error for pure SUCCESS, got %v", err)
	}

	// Case 2: Kèm theo PHP Warning giống trường hợp thực tế của người dùng
	warningOut := `PHP Warning:  rmdir(/usr/local/lsws/Example/html/wp-content/litespeed/css/): Directory not empty in /usr/local/lsws/Example/html/wp-content/plugins/litespeed-cache/src/file.cls.php on line 213
SUCCESS`
	if err := ParseApplyOutput(warningOut); err != nil {
		t.Errorf("expected nil error for SUCCESS with PHP warning, got %v", err)
	}

	// Case 3: Trả về lỗi ERROR: từ WordPress
	errorOut := `PHP Notice: some notice
ERROR: Không thể kích hoạt plugin: Plugin file does not exist.`
	err := ParseApplyOutput(errorOut)
	if err == nil || !strings.Contains(err.Error(), "Không thể kích hoạt plugin") {
		t.Errorf("expected extracted error message, got %v", err)
	}

	// Case 4: Lỗi không xác định
	unknownOut := "Fatal error: Allowed memory size of 12345 bytes exhausted"
	err = ParseApplyOutput(unknownOut)
	if err == nil || !strings.Contains(err.Error(), "phản hồi từ WordPress") {
		t.Errorf("expected raw output error, got %v", err)
	}
}

func TestParseExportOutput(t *testing.T) {
	// Case 1: Chuẩn SUCCESS kèm base64 JSON
	rawJSON := `{"cache": 1}`
	b64 := "SUCCESS:" + "eyJjYWNoZSI6IDF9"
	data, err := ParseExportOutput(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != rawJSON {
		t.Errorf("expected %s, got %s", rawJSON, string(data))
	}

	// Case 2: Kèm PHP Warning phía trước
	warningWithExport := "PHP Warning: Cannot modify header information\n" + b64
	data, err = ParseExportOutput(warningWithExport)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != rawJSON {
		t.Errorf("expected %s, got %s", rawJSON, string(data))
	}

	// Case 3: Trả về lỗi ERROR:
	errOut := "PHP Warning: bla\nERROR: Không tìm thấy cấu hình"
	_, err = ParseExportOutput(errOut)
	if err == nil || !strings.Contains(err.Error(), "Không tìm thấy cấu hình") {
		t.Errorf("expected error containing 'Không tìm thấy cấu hình', got %v", err)
	}
}
