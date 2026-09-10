package backup

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ols-cli/ols/internal/config"
)

func TestFormatBackupFilename(t *testing.T) {
	fixedTime := time.Date(2026, 9, 7, 14, 30, 0, 0, time.UTC)
	name := FormatBackupFilename("mysite.com", fixedTime)
	expected := "20260907_143000_mysite.com.tar.gz"
	if name != expected {
		t.Errorf("expected %s, got %s", expected, name)
	}
	if !strings.HasSuffix(name, ".tar.gz") {
		t.Errorf("expected tar.gz extension")
	}
}

func TestBackupAllSitesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SystemDir: tmpDir}
	bm := NewBackupManager(cfg)

	paths, errs := bm.BackupAllSites()
	if len(paths) != 0 || len(errs) != 0 {
		t.Errorf("expected empty results for empty sites, got paths=%v, errs=%v", paths, errs)
	}
}

func TestBackupAllSitesProgress(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SystemDir: tmpDir}
	bm := NewBackupManager(cfg)

	// Tạo 2 site giả lập
	site1 := filepath.Join(tmpDir, "sites", "site1.com")
	site2 := filepath.Join(tmpDir, "sites", "site2.com")
	_ = os.MkdirAll(filepath.Join(site1, "html"), 0755)
	_ = os.WriteFile(filepath.Join(site1, "html", "index.html"), []byte("site1"), 0644)
	_ = os.MkdirAll(filepath.Join(site2, "html"), 0755)
	_ = os.WriteFile(filepath.Join(site2, "html", "index.html"), []byte("site2"), 0644)

	var progressReports []string
	paths, errs := bm.BackupAllSitesProgress(func(current, total int, domain, backupPath string, err error) {
		progressReports = append(progressReports, domain)
	})

	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 backup paths, got: %v", paths)
	}
	if len(progressReports) != 2 {
		t.Fatalf("expected 2 progress reports, got: %v", progressReports)
	}
}

func TestBackupExcludesLogs(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{SystemDir: tmpDir}
	bm := NewBackupManager(cfg)

	site := filepath.Join(tmpDir, "sites", "logsite.com")
	_ = os.MkdirAll(filepath.Join(site, "html"), 0755)
	_ = os.WriteFile(filepath.Join(site, "html", "index.html"), []byte("content"), 0644)
	_ = os.MkdirAll(filepath.Join(site, "logs"), 0755)
	_ = os.WriteFile(filepath.Join(site, "logs", "access.log"), []byte("huge log data"), 0644)

	tarPath, err := bm.BackupSite("logsite.com")
	if err != nil {
		t.Fatalf("BackupSite failed: %v", err)
	}

	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == "logs" || strings.HasPrefix(hdr.Name, "logs/") {
			t.Errorf("expected backup to exclude logs directory, but found: %s", hdr.Name)
		}
	}
}

