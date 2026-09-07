package util

import "testing"

func TestGeneratePassword(t *testing.T) {
	pass1, err := GenerateRandomString(24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pass1) != 24 {
		t.Errorf("expected len 24, got %d", len(pass1))
	}

	pass2, _ := GenerateRandomString(24)
	if pass1 == pass2 {
		t.Errorf("passwords should be unique")
	}
}
