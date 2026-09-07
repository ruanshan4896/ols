package backup

import (
	"strings"
	"testing"
	"time"
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
