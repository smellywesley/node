package model

import "testing"

func TestLifecycleTransitions(t *testing.T) {
	valid := [][2]ProcessState{
		{StateCreated, StateQueued},
		{StateQueued, StateRunning},
		{StateRunning, StateWaitingApproval},
		{StateWaitingApproval, StateRunning},
		{StateRunning, StateSucceeded},
	}
	for _, transition := range valid {
		if !CanTransition(transition[0], transition[1]) {
			t.Fatalf("expected transition %s -> %s", transition[0], transition[1])
		}
	}
	if CanTransition(StateSucceeded, StateRunning) {
		t.Fatal("terminal state must not transition")
	}
}

func TestManifestValidation(t *testing.T) {
	manifest := Manifest{
		Image: "example@sha256:abc",
		Task:  "test",
		Implementation: Implementation{
			Command: []string{"agent"},
		},
	}
	manifest.ApplyDefaults()
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestManifestRejectsNegativeBudgetAndUnpricedCostLimit(t *testing.T) {
	manifest := Manifest{
		Image: "example@sha256:abc", Task: "test",
		Implementation: Implementation{Command: []string{"agent"}},
		Budget:         Budget{MaxConcurrency: 1, MaxTokens: -1},
	}
	if err := manifest.Validate(); err == nil {
		t.Fatal("negative budget should be rejected")
	}
	manifest.Budget.MaxTokens = 0
	manifest.Budget.MaxCostUSD = 1
	if err := manifest.Validate(); err == nil {
		t.Fatal("cost budget without pricing should be rejected")
	}
}

func TestManifestRejectsApprovalBypassCapabilities(t *testing.T) {
	manifest := validManifest()
	manifest.Mounts = []Mount{{Source: ".", Target: "/workspace", ReadOnly: false}}
	manifest.Capabilities.FilesystemWrite = []string{"/workspace"}
	manifest.ApprovalRules = []ApprovalRule{{Action: "fs.write"}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("approval-gated writes must not have a directly writable mount")
	}

	manifest = validManifest()
	manifest.Capabilities.NetworkDestinations = []string{"example.com"}
	manifest.ApprovalRules = []ApprovalRule{{Action: "network.connect", Match: "example.com"}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("approval-gated network access must not have direct egress")
	}

	manifest = validManifest()
	manifest.Mounts = []Mount{{Source: ".", Target: "/workspace", ReadOnly: false}}
	manifest.Capabilities.FilesystemWrite = []string{"/workspace"}
	manifest.ApprovalRules = []ApprovalRule{{Action: "fs.write", Match: "/workspace/protected"}}
	if err := manifest.Validate(); err == nil {
		t.Fatal("broad writable mount must not bypass a narrow approval subtree")
	}
}

func validManifest() Manifest {
	return Manifest{
		Image: "example@sha256:abc",
		Task:  "test",
		Implementation: Implementation{
			Command: []string{"agent"},
		},
		Budget: Budget{MaxConcurrency: 1},
		Retry:  RetryPolicy{MaxAttempts: 1},
	}
}
