package docker

import "testing"

func TestBuildComposeArgs(t *testing.T) {
	args := buildComposeArgs("/opt/ols/core", "up", "-d")
	expected := []string{"compose", "--project-directory", "/opt/ols/core", "up", "-d"}
	if len(args) != len(expected) {
		t.Fatalf("expected len %d, got %d", len(expected), len(args))
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Errorf("arg[%d] expected %s, got %s", i, expected[i], args[i])
		}
	}
}
