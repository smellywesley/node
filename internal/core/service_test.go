package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/agentos/agentos/internal/model"
	"github.com/agentos/agentos/internal/runner"
	"github.com/agentos/agentos/internal/store"
)

type blockingRunner struct {
	started chan string
	release chan struct{}
}

func (r *blockingRunner) Run(ctx context.Context, p model.Process, _ runner.ToolHandler, _ runner.UsageHandler, emit func(string, any)) (runner.Output, error) {
	r.started <- p.ID
	select {
	case <-ctx.Done():
		return runner.Output{}, ctx.Err()
	case <-r.release:
		return runner.Output{}, nil
	}
}
func (r *blockingRunner) Cancel(context.Context, string) error { return nil }

type deadlineRunner struct{}

func (deadlineRunner) Run(ctx context.Context, _ model.Process, _ runner.ToolHandler, _ runner.UsageHandler, _ func(string, any)) (runner.Output, error) {
	<-ctx.Done()
	return runner.Output{}, ctx.Err()
}
func (deadlineRunner) Cancel(context.Context, string) error { return nil }

type usageRunner struct {
	usage model.Usage
}

func (r usageRunner) Run(context.Context, model.Process, runner.ToolHandler, runner.UsageHandler, func(string, any)) (runner.Output, error) {
	return runner.Output{Usage: r.usage}, nil
}
func (usageRunner) Cancel(context.Context, string) error { return nil }

type cancellableRunner struct {
	started chan struct{}
	stop    chan struct{}
	once    sync.Once
}

type retryRunner struct {
	mu       sync.Mutex
	attempts int
}

func (r *retryRunner) Run(context.Context, model.Process, runner.ToolHandler, runner.UsageHandler, func(string, any)) (runner.Output, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts++
	if r.attempts == 1 {
		return runner.Output{Usage: model.Usage{Tokens: 7}}, errors.New("transient")
	}
	return runner.Output{Usage: model.Usage{Tokens: 9}}, nil
}
func (*retryRunner) Cancel(context.Context, string) error { return nil }

func newCancellableRunner() *cancellableRunner {
	return &cancellableRunner{started: make(chan struct{}), stop: make(chan struct{})}
}

func (r *cancellableRunner) Run(ctx context.Context, _ model.Process, _ runner.ToolHandler, _ runner.UsageHandler, _ func(string, any)) (runner.Output, error) {
	close(r.started)
	select {
	case <-ctx.Done():
		return runner.Output{}, ctx.Err()
	case <-r.stop:
		return runner.Output{}, context.Canceled
	}
}

func (r *cancellableRunner) Cancel(context.Context, string) error {
	r.once.Do(func() { close(r.stop) })
	return nil
}

func TestProcessLifecycleAndReplay(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 1), release: make(chan struct{})}
	service := New(db, fake, 1)
	defer service.Close()
	process, err := service.Create(context.Background(), testManifest())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-fake.started:
	case <-time.After(2 * time.Second):
		t.Fatal("process did not start")
	}
	close(fake.release)
	deadline := time.Now().Add(2 * time.Second)
	for {
		process, err = service.Get(context.Background(), process.ID)
		if err != nil {
			t.Fatal(err)
		}
		if process.State == model.StateSucceeded {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("process did not succeed, state=%s", process.State)
		}
		time.Sleep(10 * time.Millisecond)
	}
	replayed, err := service.Replay(context.Background(), process.ID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed != model.StateSucceeded {
		t.Fatalf("replayed state=%s", replayed)
	}
}

func TestToolApprovalIsSingleUseAndDigestBound(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 1), release: make(chan struct{})}
	service := New(db, fake, 1)
	defer service.Close()
	manifest := testManifest()
	manifest.Capabilities.Tools = []string{"fs.write"}
	manifest.Capabilities.FilesystemWrite = []string{"/workspace"}
	manifest.ApprovalRules = []model.ApprovalRule{{Action: "fs.write"}}
	process, err := service.Create(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started
	decision, err := service.RequestTool(context.Background(), process.ID, model.ToolRequest{
		IdempotencyKey: "write-1", Action: "fs.write", Resource: "/workspace/a",
		Payload: json.RawMessage(`{"content":"ok"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != "waiting_approval" {
		t.Fatalf("status=%s", decision.Status)
	}
	if _, err = service.DecideApproval(context.Background(), decision.ApprovalID, "approved", "test"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.DecideApproval(context.Background(), decision.ApprovalID, "approved", "again"); err == nil {
		t.Fatal("approval must be single use")
	}
	close(fake.release)
}

func TestChildReservationsAndNarrowing(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 1), release: make(chan struct{})}
	service := New(db, fake, 1)
	defer service.Close()
	parentManifest := testManifest()
	parentManifest.Budget.MaxTokens = 100
	parentManifest.Budget.MaxConcurrency = 2
	parentManifest.ApprovalRules = []model.ApprovalRule{{Action: "fs.write", Match: "/workspace/protected"}}
	parentManifest.Capabilities.Tools = []string{"fs.write"}
	parentManifest.Capabilities.FilesystemWrite = []string{"/workspace"}
	parent, err := service.Create(context.Background(), parentManifest)
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started

	child := parentManifest
	child.ParentID = parent.ID
	child.Name = "child"
	child.Budget.MaxTokens = 60
	child.Budget.MaxConcurrency = 1
	if _, err = service.Create(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Create(context.Background(), child); err == nil {
		t.Fatal("sibling reservations should not exceed parent budget")
	}
	widened := child
	widened.Budget.MaxTokens = 10
	widened.Image = "other@sha256:def"
	if _, err = service.Create(context.Background(), widened); err == nil {
		t.Fatal("child runtime image must not widen parent")
	}
	childWithoutApproval := child
	childWithoutApproval.Name = "unsafe-child"
	childWithoutApproval.ApprovalRules = nil
	childWithoutApproval.Budget.MaxTokens = 10
	childWithoutApproval.Budget.MaxConcurrency = 1
	if _, err = service.Create(context.Background(), childWithoutApproval); err == nil {
		t.Fatal("child must not drop parent approval rules")
	}
	close(fake.release)
}

func TestToolCompletionIsSingleUse(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 1), release: make(chan struct{})}
	service := New(db, fake, 1)
	defer service.Close()
	manifest := testManifest()
	manifest.Capabilities.Tools = []string{"pure.echo"}
	process, err := service.Create(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started
	_, err = service.RequestTool(context.Background(), process.ID, model.ToolRequest{
		IdempotencyKey: "once", Action: "pure.echo", Payload: json.RawMessage(`{"x":1}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = service.StartTool(context.Background(), process.ID, "once"); err != nil {
		t.Fatal(err)
	}
	result := model.ToolResult{Status: "completed", Output: json.RawMessage(`{"ok":true}`)}
	if err = service.CompleteTool(context.Background(), process.ID, "once", result); err != nil {
		t.Fatal(err)
	}
	if err = service.CompleteTool(context.Background(), process.ID, "once", result); err == nil {
		t.Fatal("second completion should fail")
	}
	close(fake.release)
}

func TestRuntimeDurationAndUsageBudgetsFailProcess(t *testing.T) {
	t.Run("duration", func(t *testing.T) {
		db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		service := New(db, deadlineRunner{}, 1)
		defer service.Close()
		manifest := testManifest()
		manifest.Budget.MaxDurationSec = 1
		process, err := service.Create(context.Background(), manifest)
		if err != nil {
			t.Fatal(err)
		}
		waitForState(t, service, process.ID, model.StateFailed, 3*time.Second)
	})

	t.Run("tokens", func(t *testing.T) {
		db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		service := New(db, usageRunner{usage: model.Usage{Tokens: 101}}, 1)
		defer service.Close()
		process, err := service.Create(context.Background(), testManifest())
		if err != nil {
			t.Fatal(err)
		}
		waitForState(t, service, process.ID, model.StateFailed, 2*time.Second)
		process, err = service.Get(context.Background(), process.ID)
		if err != nil {
			t.Fatal(err)
		}
		if process.Usage.Tokens != 101 {
			t.Fatalf("over-limit usage was not charged: %+v", process.Usage)
		}
	})
}

func TestInheritedUsageAndRecursiveCancellation(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 1), release: make(chan struct{})}
	service := New(db, fake, 1)
	defer service.Close()
	parentManifest := testManifest()
	parentManifest.Budget.MaxTokens = 100
	parentManifest.Budget.MaxChildren = 2
	parent, err := service.Create(context.Background(), parentManifest)
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started

	childManifest := parentManifest
	childManifest.ParentID = parent.ID
	childManifest.Name = "child"
	childManifest.Budget.MaxTokens = 100
	child, err := service.Create(context.Background(), childManifest)
	if err != nil {
		t.Fatal(err)
	}
	grandchildManifest := childManifest
	grandchildManifest.ParentID = child.ID
	grandchildManifest.Name = "grandchild"
	grandchildManifest.Budget.MaxTokens = 100
	grandchild, err := service.Create(context.Background(), grandchildManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.UpdateUsage(context.Background(), child.ID, model.Usage{Tokens: 60}); err != nil {
		t.Fatal(err)
	}
	if err = service.UpdateUsage(context.Background(), grandchild.ID, model.Usage{Tokens: 41}); err == nil {
		t.Fatal("descendant usage should not exceed the root budget")
	}

	if _, err = service.Transition(context.Background(), parent.ID, model.StateCancelled); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{parent.ID, child.ID, grandchild.ID} {
		process, getErr := service.Get(context.Background(), id)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if process.State != model.StateCancelled {
			t.Fatalf("%s state=%s", id, process.State)
		}
	}
	close(fake.release)
}

func TestSuspendWinsRaceWithWorkerFinalization(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := newCancellableRunner()
	service := New(db, fake, 1)
	defer service.Close()
	process, err := service.Create(context.Background(), testManifest())
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started
	if _, err = service.Transition(context.Background(), process.ID, model.StateSuspended); err != nil {
		t.Fatal(err)
	}
	waitForState(t, service, process.ID, model.StateSuspended, time.Second)
}

func TestCloseLeavesRunningProcessRecoverable(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := newCancellableRunner()
	service := New(db, fake, 1)
	process, err := service.Create(context.Background(), testManifest())
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started
	service.Close()
	process, err = db.GetProcess(context.Background(), process.ID)
	if err != nil {
		t.Fatal(err)
	}
	if process.State != model.StateRunning {
		t.Fatalf("state=%s, want recoverable running", process.State)
	}
}

func TestApprovalGatesBrokeredFilesystemWrite(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("AGENTOS_WORKSPACE_ROOT", workspace)
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 1), release: make(chan struct{})}
	service := New(db, fake, 1)
	defer service.Close()
	manifest := testManifest()
	manifest.Implementation.Adapter = "jsonlines-tools"
	manifest.Capabilities.Tools = []string{"fs.write"}
	manifest.Capabilities.FilesystemRead = []string{"/workspace"}
	manifest.Capabilities.FilesystemWrite = []string{"/workspace"}
	manifest.Mounts = []model.Mount{{Source: workspace, Target: "/workspace", ReadOnly: true}}
	manifest.ApprovalRules = []model.ApprovalRule{{Action: "fs.write"}}
	process, err := service.Create(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started

	resultChannel := make(chan model.ToolResult, 1)
	go func() {
		resultChannel <- service.handleWorkerTool(context.Background(), process, model.ToolRequest{
			IdempotencyKey: "write-1",
			Action:         "fs.write",
			Resource:       "/workspace/reviewed.txt",
			Payload:        json.RawMessage(`{"content":"approved artifact"}`),
		})
	}()

	var approval model.Approval
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		approvals, approvalErr := service.Approvals(context.Background())
		if approvalErr != nil {
			t.Fatal(approvalErr)
		}
		if len(approvals) == 1 {
			approval = approvals[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if approval.ID == "" {
		t.Fatal("approval was not created")
	}
	if _, statErr := os.Stat(filepath.Join(workspace, "reviewed.txt")); !os.IsNotExist(statErr) {
		t.Fatal("artifact was written before approval")
	}
	if _, err = service.DecideApproval(context.Background(), approval.ID, "approved", "reviewed"); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-resultChannel:
		if result.Status != "completed" {
			t.Fatalf("result=%+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("brokered write did not resume after approval")
	}
	content, err := os.ReadFile(filepath.Join(workspace, "reviewed.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "approved artifact" {
		t.Fatalf("content=%q", content)
	}
	close(fake.release)
}

func TestCancelledProcessCannotCompletePendingApproval(t *testing.T) {
	workspace := t.TempDir()
	t.Setenv("AGENTOS_WORKSPACE_ROOT", workspace)
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 1), release: make(chan struct{})}
	service := New(db, fake, 1)
	defer service.Close()
	manifest := testManifest()
	manifest.Capabilities.Tools = []string{"fs.write"}
	manifest.Capabilities.FilesystemRead = []string{"/workspace"}
	manifest.Capabilities.FilesystemWrite = []string{"/workspace"}
	manifest.Mounts = []model.Mount{{Source: workspace, Target: "/workspace", ReadOnly: true}}
	manifest.ApprovalRules = []model.ApprovalRule{{Action: "fs.write"}}
	process, err := service.Create(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started
	decision, err := service.RequestTool(context.Background(), process.ID, model.ToolRequest{
		IdempotencyKey: "cancelled-write", Action: "fs.write", Resource: "/workspace/cancelled.txt",
		Payload: json.RawMessage(`{"content":"must not exist"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Transition(context.Background(), process.ID, model.StateCancelled); err != nil {
		t.Fatal(err)
	}
	if _, err = service.DecideApproval(context.Background(), decision.ApprovalID, "approved", "too late"); err == nil {
		t.Fatal("terminal process approval should be rejected")
	}
	if _, statErr := os.Stat(filepath.Join(workspace, "cancelled.txt")); !os.IsNotExist(statErr) {
		t.Fatal("cancelled process wrote an artifact")
	}
	close(fake.release)
}

func TestRetryBackoffPreservesIdentityAndFailedAttemptUsage(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &retryRunner{}
	service := New(db, fake, 1)
	defer service.Close()
	manifest := testManifest()
	manifest.Retry.MaxAttempts = 2
	process, err := service.Create(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	waitForState(t, service, process.ID, model.StateSucceeded, 2*time.Second)
	process, err = service.Get(context.Background(), process.ID)
	if err != nil {
		t.Fatal(err)
	}
	if process.Attempt != 2 {
		t.Fatalf("attempt=%d", process.Attempt)
	}
	if process.Usage.Tokens != 9 {
		t.Fatalf("usage=%+v", process.Usage)
	}
	events, err := service.Events(context.Background(), process.ID)
	if err != nil {
		t.Fatal(err)
	}
	var retried bool
	for _, event := range events {
		if event.Type == "process.retry_scheduled" {
			retried = true
		}
	}
	if !retried {
		t.Fatal("retry event was not recorded")
	}
}

func TestHierarchyConcurrencyDelaysChildAdmission(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fake := &blockingRunner{started: make(chan string, 2), release: make(chan struct{})}
	service := New(db, fake, 2)
	defer service.Close()
	parentManifest := testManifest()
	parentManifest.Budget.MaxConcurrency = 1
	parent, err := service.Create(context.Background(), parentManifest)
	if err != nil {
		t.Fatal(err)
	}
	<-fake.started
	childManifest := parentManifest
	childManifest.ParentID = parent.ID
	childManifest.Name = "child"
	childManifest.Budget.MaxChildren = 0
	child, err := service.Create(context.Background(), childManifest)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case started := <-fake.started:
		t.Fatalf("child %s started while parent consumed hierarchy concurrency", started)
	case <-time.After(250 * time.Millisecond):
	}
	close(fake.release)
	waitForState(t, service, parent.ID, model.StateSucceeded, 2*time.Second)
	waitForState(t, service, child.ID, model.StateSucceeded, 2*time.Second)
}

func waitForState(t *testing.T, service *Service, id string, expected model.ProcessState, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		process, err := service.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if process.State == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	process, err := service.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("process state=%s, want %s", process.State, expected)
}

func testManifest() model.Manifest {
	return model.Manifest{
		Name: "test", Image: "example@sha256:abc", Task: "test",
		Implementation: model.Implementation{Command: []string{"agent"}},
		Budget:         model.Budget{MaxConcurrency: 1, MaxChildren: 2, MaxTokens: 100},
	}
}
