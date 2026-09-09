package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestMenuExit(t *testing.T) {
	in := strings.NewReader("0\n")
	var out bytes.Buffer

	err := RunInteractiveMenu(in, &out)
	if err != nil {
		t.Fatalf("mong đợi không có lỗi, nhận: %v", err)
	}
}

func TestMenuInvalidThenExit(t *testing.T) {
	in := strings.NewReader("99\n\n0\n")
	var out bytes.Buffer

	err := RunInteractiveMenu(in, &out)
	if err != nil {
		t.Fatalf("mong đợi không có lỗi, nhận: %v", err)
	}
}

func TestPrintMenu(t *testing.T) {
	var buf bytes.Buffer
	PrintMenu(&buf)
	output := buf.String()

	if !strings.Contains(output, "OLS-CLI") {
		t.Errorf("expected menu to contain OLS-CLI header")
	}
	if !strings.Contains(output, "HẠ TẦNG CỐT LÕI") {
		t.Errorf("expected menu to contain HẠ TẦNG CỐT LÕI")
	}
	if !strings.Contains(output, "QUẢN LÝ WEBSITE") {
		t.Errorf("expected menu to contain QUẢN LÝ WEBSITE")
	}
	if !strings.Contains(output, "[13]") {
		t.Errorf("expected menu to contain [13]")
	}
	if !strings.Contains(output, "[14]") {
		t.Errorf("expected menu to contain [14]")
	}
	if !strings.Contains(output, "[15]") {
		t.Errorf("expected menu to contain [15]")
	}
}
