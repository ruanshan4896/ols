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
