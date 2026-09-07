package cmd

import (
	"bytes"
	"testing"
)

func TestPMACmdHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"pma", "--help"})
	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing pma help: %v", err)
	}
}
