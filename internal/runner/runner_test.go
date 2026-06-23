package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentos/agentos/internal/model"
)

func TestSecureHostPathStaysUnderWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "workspace")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := secureHostPath(child, root)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != child {
		t.Fatalf("resolved=%q", resolved)
	}
	if _, err = secureHostPath(filepath.Dir(root), root); err == nil {
		t.Fatal("path outside workspace root should be rejected")
	}
}

func TestMountMustMatchCapabilityAndRedactsSecrets(t *testing.T) {
	capabilities := model.Capabilities{
		FilesystemRead:  []string{"/workspace"},
		FilesystemWrite: []string{"/workspace/out"},
	}
	if !mountCovered(model.Mount{Target: "/workspace/input", ReadOnly: true}, capabilities) {
		t.Fatal("read-only mount should match read capability")
	}
	if mountCovered(model.Mount{Target: "/workspace/input", ReadOnly: false}, capabilities) {
		t.Fatal("writable mount outside write capability should be rejected")
	}
	if got := redactSecrets("token=super-secret", []string{"super-secret"}); got != "token=[REDACTED]" {
		t.Fatalf("redacted=%q", got)
	}
}

func TestReplaceNetworkSupportsDockerArgumentForms(t *testing.T) {
	equals := replaceNetwork([]string{"run", "--network=none", "image"}, "isolated")
	if equals[1] != "--network=isolated" {
		t.Fatalf("equals form=%v", equals)
	}

	separate := replaceNetwork([]string{"run", "--network", "none", "image"}, "isolated")
	if separate[2] != "isolated" {
		t.Fatalf("separate form=%v", separate)
	}
}
