//go:build windows

package state

import (
	"os/exec"
	"strings"
	"testing"
)

func TestEnsureDirRemovesBroadWindowsPrincipals(t *testing.T) {
	directory := t.TempDir()
	if err := EnsureDir(directory); err != nil {
		t.Fatal(err)
	}
	if err := CheckDirPrivate(directory); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("icacls", directory).CombinedOutput()
	if err != nil {
		t.Fatalf("read ACL: %v: %s", err, output)
	}
	if containsBroadPrincipal(string(output)) {
		t.Fatalf("state directory ACL still contains a broad principal:\n%s", strings.TrimSpace(string(output)))
	}
}
