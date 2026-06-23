//go:build windows

package api

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureTokenDoesNotRewriteParentACL(t *testing.T) {
	directory := t.TempDir()
	before, err := exec.Command("icacls", directory).CombinedOutput()
	if err != nil {
		t.Fatalf("read ACL before token creation: %v: %s", err, before)
	}
	if _, err = EnsureToken(filepath.Join(directory, "token")); err != nil {
		t.Fatal(err)
	}
	after, err := exec.Command("icacls", directory).CombinedOutput()
	if err != nil {
		t.Fatalf("read ACL after token creation: %v: %s", err, after)
	}
	if normalizeACL(string(before)) != normalizeACL(string(after)) {
		t.Fatalf("parent directory ACL changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func normalizeACL(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}
