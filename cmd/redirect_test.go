package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRedirectCmdHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"redirect", "--help"})

	err := RootCmd.Execute()
	if err != nil {
		t.Fatalf("kỳ vọng không có lỗi, nhận: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "add") {
		t.Errorf("kỳ vọng help chứa lệnh 'add'")
	}
	if !strings.Contains(out, "list") {
		t.Errorf("kỳ vọng help chứa lệnh 'list'")
	}
	if !strings.Contains(out, "remove") {
		t.Errorf("kỳ vọng help chứa lệnh 'remove'")
	}
	if !strings.Contains(out, "test") {
		t.Errorf("kỳ vọng help chứa lệnh 'test'")
	}
}
