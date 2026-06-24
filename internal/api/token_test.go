package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotateTokenReplacesExistingToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	first, err := EnsureToken(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RotateToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("rotated token matched original token")
	}
	if len(second) != 64 {
		t.Fatalf("unexpected token length: %d", len(second))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != second {
		t.Fatal("token file did not contain rotated token")
	}
}
