package cmd

import (
	"testing"

	"github.com/ols-cli/ols/internal/shield"
)

func TestFormatStatus(t *testing.T) {
	enabled := formatStatus(true)
	if enabled == "" {
		t.Errorf("expected formatted status for true")
	}

	disabled := formatStatus(false)
	if disabled == "" {
		t.Errorf("expected formatted status for false")
	}
}

func TestPrintShieldReport(t *testing.T) {
	cfg := shield.DefaultShieldConfig()
	// Just ensure PrintShieldReport runs without panic
	PrintShieldReport("test.local", cfg)
}
