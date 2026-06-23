package policy

import (
	"encoding/json"
	"testing"

	"github.com/agentos/agentos/internal/model"
)

func TestEvaluateDefaultDenyAndApproval(t *testing.T) {
	manifest := model.Manifest{
		Capabilities: model.Capabilities{
			Tools:           []string{"fs.read", "fs.write"},
			FilesystemRead:  []string{"/workspace"},
			FilesystemWrite: []string{"/workspace/out"},
		},
		ApprovalRules: []model.ApprovalRule{{Action: "fs.write", Match: "release"}},
	}
	allowed, approval, _ := Evaluate(manifest, model.ToolRequest{
		Action: "fs.read", Resource: "/workspace/main.go",
	})
	if !allowed || approval {
		t.Fatal("declared read should be authorized without approval")
	}
	allowed, _, _ = Evaluate(manifest, model.ToolRequest{
		Action: "fs.read", Resource: "/etc/passwd",
	})
	if allowed {
		t.Fatal("out-of-scope path should be denied")
	}
	allowed, approval, _ = Evaluate(manifest, model.ToolRequest{
		Action: "fs.write", Resource: "/workspace/out/release.txt",
	})
	if !allowed || !approval {
		t.Fatal("release write should require approval")
	}
}

func TestDigestBindsPayload(t *testing.T) {
	first, err := Digest(model.ActionEnvelope{ProcessID: "p", Payload: json.RawMessage(`{"value":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Digest(model.ActionEnvelope{ProcessID: "p", Payload: json.RawMessage(`{"value":2}`)})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("different payloads must produce different approval digests")
	}
}
