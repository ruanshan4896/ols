package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"version"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing version command: %v", err)
	}

	output := buf.String()
	expected := "ols-cli version 0.1.0\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}
