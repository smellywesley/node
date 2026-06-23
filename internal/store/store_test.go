package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentos/agentos/internal/model"
)

func TestRecoveryRequeuesRunningProcess(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	p := model.Process{
		ID: "p1", Name: "test", State: model.StateCreated, CreatedAt: now, UpdatedAt: now,
		Manifest: model.Manifest{Image: "test", Task: "test", Implementation: model.Implementation{Command: []string{"test"}}},
	}
	if err = s.CreateProcess(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Transition(ctx, p.ID, model.StateQueued, "process.queued", nil); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Transition(ctx, p.ID, model.StateRunning, "process.running", nil); err != nil {
		t.Fatal(err)
	}
	ids, err := s.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != p.ID {
		t.Fatalf("unexpected recovered processes: %v", ids)
	}
	recovered, err := s.GetProcess(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.State != model.StateQueued {
		t.Fatalf("expected queued, got %s", recovered.State)
	}
}

func TestRecoveryReturnsAlreadyQueuedProcess(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	p := model.Process{
		ID: "p1", Name: "test", State: model.StateCreated, CreatedAt: now, UpdatedAt: now,
		Manifest: model.Manifest{Image: "test", Task: "test", Implementation: model.Implementation{Command: []string{"test"}}},
	}
	if err = s.CreateProcess(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Transition(ctx, p.ID, model.StateQueued, "process.queued", nil); err != nil {
		t.Fatal(err)
	}
	ids, err := s.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != p.ID {
		t.Fatalf("queued process not recovered: %v", ids)
	}
}

func TestIdempotencyKeyRejectsChangedRequest(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	p := model.Process{
		ID: "p1", Name: "test", State: model.StateCreated, CreatedAt: now, UpdatedAt: now,
		Manifest: model.Manifest{Image: "test", Task: "test", Implementation: model.Implementation{Command: []string{"test"}}},
	}
	if err = s.CreateProcess(ctx, p); err != nil {
		t.Fatal(err)
	}
	created, err := s.CreateToolCall(ctx, p.ID, "same", "fs.write", "hash-a", map[string]string{"a": "b"})
	if err != nil || !created {
		t.Fatalf("first create: created=%v err=%v", created, err)
	}
	created, err = s.CreateToolCall(ctx, p.ID, "same", "fs.write", "hash-b", map[string]string{"a": "c"})
	if err != nil || created {
		t.Fatalf("duplicate create: created=%v err=%v", created, err)
	}
	hash, _, _, err := s.ToolCall(ctx, p.ID, "same")
	if err != nil {
		t.Fatal(err)
	}
	if hash != "hash-a" {
		t.Fatalf("original request hash changed to %q", hash)
	}
}

func TestAuditRecordsIncludeApprovalsAndToolCalls(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	p := model.Process{
		ID: "p1", Name: "test", State: model.StateCreated, CreatedAt: now, UpdatedAt: now,
		Manifest: model.Manifest{Image: "test", Task: "test", Implementation: model.Implementation{Command: []string{"test"}}},
	}
	if err = s.CreateProcess(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateToolCall(ctx, p.ID, "key-1", "fs.write", "hash-1", map[string]string{"secret": "value"}); err != nil {
		t.Fatal(err)
	}
	approval := model.Approval{
		ID: "approval-1", ProcessID: p.ID, IdempotencyKey: "key-1", Action: "fs.write",
		Payload: json.RawMessage(`{"secret":"value"}`), Status: "pending", CreatedAt: now,
	}
	if err = s.CreateApproval(ctx, approval, "hash-1", "v1"); err != nil {
		t.Fatal(err)
	}
	approvals, err := s.ApprovalsForProcess(ctx, p.ID)
	if err != nil || len(approvals) != 1 || approvals[0].ID != approval.ID {
		t.Fatalf("approvals=%v err=%v", approvals, err)
	}
	calls, err := s.ToolCallsForProcess(ctx, p.ID)
	if err != nil || len(calls) != 1 || calls[0].RequestHash != "hash-1" {
		t.Fatalf("calls=%v err=%v", calls, err)
	}
}

func TestLatestCheckpointUsesNewestAppendOnlyEvent(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	process := model.Process{
		ID: "p1", Name: "checkpoint", State: model.StateCreated, CreatedAt: now, UpdatedAt: now,
		Manifest: model.Manifest{Image: "test", Task: "test", Implementation: model.Implementation{Command: []string{"test"}}},
	}
	if err = s.CreateProcess(ctx, process); err != nil {
		t.Fatal(err)
	}
	if err = s.AppendEvent(ctx, process.ID, "process.checkpoint", map[string]any{"step": 1}); err != nil {
		t.Fatal(err)
	}
	if err = s.AppendEvent(ctx, process.ID, "process.checkpoint", map[string]any{"step": 2}); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := s.LatestCheckpoint(ctx, process.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(checkpoint) != `{"step":2}` {
		t.Fatalf("checkpoint=%s", checkpoint)
	}
}
