package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/agentos/agentos/internal/model"
	"github.com/agentos/agentos/internal/policy"
	"github.com/agentos/agentos/internal/runner"
	"github.com/agentos/agentos/internal/store"
)

type Service struct {
	store       *store.Store
	runner      runner.Runner
	queue       chan string
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	createMu    sync.Mutex
	budgetMu    sync.Mutex
	lifecycleMu sync.Mutex
	scheduleMu  sync.Mutex
	stateMu     sync.Mutex
	stopping    bool
	active      sync.Map
}

func New(store *store.Store, processRunner runner.Runner, concurrency int) *Service {
	if concurrency < 1 {
		concurrency = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		store: store, runner: processRunner, queue: make(chan string, 256),
		ctx: ctx, cancel: cancel,
	}
	for i := 0; i < concurrency; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	return s
}

func (s *Service) Close() {
	s.stateMu.Lock()
	s.stopping = true
	s.stateMu.Unlock()
	s.cancel()
	processes, err := s.store.ListProcesses(context.Background())
	if err == nil {
		for _, p := range processes {
			if p.State == model.StateRunning || p.State == model.StateWaitingApproval {
				_ = s.runner.Cancel(context.Background(), p.ID)
			}
		}
	}
	s.wg.Wait()
}

func (s *Service) Recover(ctx context.Context) error {
	ids, err := s.store.Recover(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		p, getErr := s.store.GetProcess(ctx, id)
		if getErr != nil {
			continue
		}
		if cancelErr := s.runner.Cancel(ctx, id); cancelErr != nil {
			_ = s.store.SetFailure(ctx, id, "recovery could not fence stale container: "+cancelErr.Error())
			_, _ = s.store.Transition(ctx, id, model.StateFailed, "process.recovery_failed", map[string]any{"error": cancelErr.Error()})
			continue
		}
		if p.State == model.StateQueued {
			s.enqueue(id)
		}
	}
	return nil
}

func (s *Service) Create(ctx context.Context, manifest model.Manifest) (model.Process, error) {
	s.createMu.Lock()
	defer s.createMu.Unlock()
	manifest.ApplyDefaults()
	if err := manifest.Validate(); err != nil {
		return model.Process{}, err
	}
	if manifest.ParentID != "" {
		if err := s.validateChild(ctx, manifest); err != nil {
			return model.Process{}, err
		}
	}
	now := time.Now().UTC()
	p := model.Process{
		ID: uuid.NewString(), ParentID: manifest.ParentID, Name: manifest.Name,
		State: model.StateCreated, Manifest: manifest, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.CreateProcess(ctx, p); err != nil {
		return model.Process{}, err
	}
	p, err := s.store.Transition(ctx, p.ID, model.StateQueued, "process.queued", nil)
	if err != nil {
		return model.Process{}, err
	}
	s.enqueue(p.ID)
	return p, nil
}

func (s *Service) Get(ctx context.Context, id string) (model.Process, error) {
	return s.store.GetProcess(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]model.Process, error) {
	return s.store.ListProcesses(ctx)
}

func (s *Service) Events(ctx context.Context, id string) ([]model.Event, error) {
	if _, err := s.store.GetProcess(ctx, id); err != nil {
		return nil, err
	}
	return s.store.Events(ctx, id)
}

func (s *Service) Transition(ctx context.Context, id string, to model.ProcessState) (model.Process, error) {
	s.lifecycleMu.Lock()
	p, err := s.store.GetProcess(ctx, id)
	if err != nil {
		s.lifecycleMu.Unlock()
		return p, err
	}
	switch to {
	case model.StateSuspended:
		if p.State == model.StateRunning || p.State == model.StateWaitingApproval {
			if err = s.runner.Cancel(ctx, id); err != nil {
				s.lifecycleMu.Unlock()
				return p, fmt.Errorf("stop container before suspend: %w", err)
			}
		}
		p, err = s.store.Transition(ctx, id, to, "process.suspended", nil)
		s.lifecycleMu.Unlock()
		return p, err
	case model.StateQueued:
		p, err = s.store.Transition(ctx, id, to, "process.resumed", nil)
		s.lifecycleMu.Unlock()
		if err == nil {
			s.enqueue(id)
		}
		return p, err
	case model.StateCancelled:
		if p.State == model.StateRunning || p.State == model.StateWaitingApproval {
			if err = s.runner.Cancel(ctx, id); err != nil {
				s.lifecycleMu.Unlock()
				return p, fmt.Errorf("stop container before cancel: %w", err)
			}
		}
		p, err = s.store.Transition(ctx, id, to, "process.cancelled", nil)
		s.lifecycleMu.Unlock()
		if err != nil {
			return p, err
		}
		children, listErr := s.store.ListProcesses(ctx)
		if listErr != nil {
			return p, fmt.Errorf("list descendants for cancellation: %w", listErr)
		}
		for _, child := range children {
			if child.ParentID == id && !child.State.Terminal() {
				if _, childErr := s.Transition(ctx, child.ID, model.StateCancelled); childErr != nil {
					return p, fmt.Errorf("cancel child %s: %w", child.ID, childErr)
				}
			}
		}
		return p, nil
	default:
		s.lifecycleMu.Unlock()
		return p, fmt.Errorf("unsupported requested transition to %s", to)
	}
}

func (s *Service) RequestTool(ctx context.Context, processID string, req model.ToolRequest) (model.ActionDecision, error) {
	if req.IdempotencyKey == "" {
		return model.ActionDecision{}, errors.New("idempotency_key is required")
	}
	p, err := s.store.GetProcess(ctx, processID)
	if err != nil {
		return model.ActionDecision{}, err
	}
	if p.State != model.StateRunning && p.State != model.StateWaitingApproval {
		return model.ActionDecision{}, fmt.Errorf("tool requests require a running process, current state is %s", p.State)
	}
	envelope := model.ActionEnvelope{
		ProcessID: processID, IdempotencyKey: req.IdempotencyKey, Action: req.Action,
		Resource: req.Resource, Payload: req.Payload, Image: p.Manifest.Image, PolicyVersion: policy.Version,
	}
	digest, err := policy.Digest(envelope)
	if err != nil {
		return model.ActionDecision{}, err
	}
	created, err := s.store.CreateToolCall(ctx, processID, req.IdempotencyKey, req.Action, digest, envelope)
	if err != nil {
		return model.ActionDecision{}, err
	}
	if !created {
		existingHash, status, _, getErr := s.store.ToolCall(ctx, processID, req.IdempotencyKey)
		if getErr != nil {
			return model.ActionDecision{}, getErr
		}
		if existingHash != digest {
			return model.ActionDecision{}, errors.New("idempotency key was reused with different action parameters")
		}
		decision := model.ActionDecision{Status: status, Digest: digest}
		if status == "waiting_approval" {
			if approval, approvalErr := s.store.ApprovalForTool(ctx, processID, req.IdempotencyKey); approvalErr == nil {
				decision.ApprovalID = approval.ID
			}
		}
		return decision, nil
	}
	allowed, approvalRequired, reason := policy.Evaluate(p.Manifest, req)
	if !allowed {
		if err = s.store.UpdateToolCallWithEvent(ctx, processID, req.IdempotencyKey, "denied",
			map[string]any{"reason": reason}, "tool.denied",
			map[string]any{"action": req.Action, "reason": reason, "digest": digest}); err != nil {
			return model.ActionDecision{}, err
		}
		return model.ActionDecision{Status: "denied", Digest: digest, Reason: reason}, nil
	}
	if approvalRequired {
		a := model.Approval{
			ID: uuid.NewString(), ProcessID: processID, IdempotencyKey: req.IdempotencyKey,
			Action: req.Action, Resource: req.Resource, ActionDigest: digest,
			Payload: req.Payload, Status: "pending", CreatedAt: time.Now().UTC(),
		}
		if err = s.store.CreateApproval(ctx, a, digest, policy.Version); err != nil {
			return model.ActionDecision{}, err
		}
		return model.ActionDecision{Status: "waiting_approval", ApprovalID: a.ID, Digest: digest}, nil
	}
	if err = s.store.UpdateToolCallWithEvent(ctx, processID, req.IdempotencyKey, "authorized", nil,
		"tool.authorized", map[string]any{"action": req.Action, "digest": digest}); err != nil {
		return model.ActionDecision{}, err
	}
	return model.ActionDecision{Status: "authorized", Digest: digest}, nil
}

func (s *Service) DecideApproval(ctx context.Context, id, decision, reason string) (model.Approval, error) {
	if decision != "approved" && decision != "denied" {
		return model.Approval{}, errors.New("decision must be approved or denied")
	}
	a, digest, version, err := s.store.GetApproval(ctx, id)
	if err != nil {
		return a, err
	}
	if version != policy.Version {
		return a, errors.New("approval policy version is stale")
	}
	process, err := s.store.GetProcess(ctx, a.ProcessID)
	if err != nil {
		return a, err
	}
	if process.State.Terminal() {
		return a, fmt.Errorf("cannot decide approval for terminal process in state %s", process.State)
	}
	ok, err := s.store.DecideApproval(ctx, id, "pending", decision, reason)
	if err != nil {
		return a, err
	}
	if !ok {
		return a, errors.New("approval has already been decided")
	}
	a.Status = decision
	now := time.Now().UTC()
	a.DecidedAt = &now
	_ = digest
	if decision == "approved" {
		if _, workerActive := s.active.Load(a.ProcessID); !workerActive {
			s.lifecycleMu.Lock()
			process, transitionErr := s.store.Transition(ctx, a.ProcessID, model.StateQueued,
				"process.approval_recovered", map[string]any{"approval_id": id})
			s.lifecycleMu.Unlock()
			if transitionErr != nil {
				return a, transitionErr
			}
			s.enqueue(process.ID)
		}
	}
	return a, nil
}

func (s *Service) Approvals(ctx context.Context) ([]model.Approval, error) {
	return s.store.PendingApprovals(ctx)
}

func (s *Service) AuditRecords(ctx context.Context, processID string) ([]model.Approval, []model.ToolCall, error) {
	approvals, err := s.store.ApprovalsForProcess(ctx, processID)
	if err != nil {
		return nil, nil, err
	}
	toolCalls, err := s.store.ToolCallsForProcess(ctx, processID)
	return approvals, toolCalls, err
}

func (s *Service) StartTool(ctx context.Context, processID, key string) error {
	started, err := s.store.ClaimToolCallWithEvent(ctx, processID, key)
	if err != nil {
		return err
	}
	if !started {
		return errors.New("tool call is not authorized or was already started")
	}
	return nil
}

func (s *Service) CompleteTool(ctx context.Context, processID, key string, result model.ToolResult) error {
	_, status, _, err := s.store.ToolCall(ctx, processID, key)
	if err != nil {
		return err
	}
	if status != "started" {
		return fmt.Errorf("tool call cannot complete from status %s", status)
	}
	finalStatus := "completed"
	if result.Status == "outcome_unknown" {
		finalStatus = "outcome_unknown"
	} else if result.Error != "" || result.Status == "failed" {
		finalStatus = "failed"
	}
	completed, err := s.store.CompleteToolCall(ctx, processID, key, finalStatus, result)
	if err != nil {
		return err
	}
	if !completed {
		return errors.New("tool call was completed concurrently")
	}
	return nil
}

func (s *Service) UpdateUsage(ctx context.Context, processID string, usage model.Usage) error {
	s.budgetMu.Lock()
	defer s.budgetMu.Unlock()
	p, err := s.store.GetProcess(ctx, processID)
	if err != nil {
		return err
	}
	if usage.Tokens < p.Usage.Tokens || usage.CostUSD < p.Usage.CostUSD {
		return errors.New("usage counters cannot decrease")
	}
	if usage.Tokens == p.Usage.Tokens && usage.CostUSD == p.Usage.CostUSD {
		return nil
	}
	var violation error
	if p.Manifest.Budget.MaxTokens > 0 && usage.Tokens > p.Manifest.Budget.MaxTokens {
		violation = errors.New("token budget exceeded")
	}
	if p.Manifest.Budget.MaxCostUSD > 0 && usage.CostUSD > p.Manifest.Budget.MaxCostUSD {
		violation = errors.New("cost budget exceeded")
	}
	ancestors, err := s.store.Ancestors(ctx, processID)
	if err != nil {
		return err
	}
	delta := model.Usage{Tokens: usage.Tokens - p.Usage.Tokens, CostUSD: usage.CostUSD - p.Usage.CostUSD}
	for _, ancestor := range ancestors {
		total, usageErr := s.store.DescendantUsage(ctx, ancestor.ID)
		if usageErr != nil {
			return usageErr
		}
		if ancestor.Manifest.Budget.MaxTokens > 0 && total.Tokens+delta.Tokens > ancestor.Manifest.Budget.MaxTokens {
			violation = fmt.Errorf("token usage exceeds ancestor %s budget", ancestor.ID)
		}
		if ancestor.Manifest.Budget.MaxCostUSD > 0 && total.CostUSD+delta.CostUSD > ancestor.Manifest.Budget.MaxCostUSD {
			violation = fmt.Errorf("cost usage exceeds ancestor %s budget", ancestor.ID)
		}
	}
	if err = s.store.UpdateUsage(ctx, processID, usage); err != nil {
		return err
	}
	return violation
}

func (s *Service) Replay(ctx context.Context, id string) (model.ProcessState, error) {
	if _, err := s.store.GetProcess(ctx, id); err != nil {
		return "", err
	}
	events, err := s.store.Events(ctx, id)
	if err != nil {
		return "", err
	}
	state := model.StateCreated
	for _, event := range events {
		var data map[string]any
		_ = json.Unmarshal(event.Data, &data)
		switch event.Type {
		case "process.queued", "process.recovered", "process.resumed":
			if event.Type == "process.recovered" && data["to"] == string(model.StateWaitingApproval) {
				state = model.StateWaitingApproval
			} else {
				state = model.StateQueued
			}
		case "process.running", "process.approval_resumed":
			state = model.StateRunning
		case "process.waiting_approval":
			state = model.StateWaitingApproval
		case "process.suspended":
			state = model.StateSuspended
		case "process.succeeded":
			state = model.StateSucceeded
		case "process.failed":
			state = model.StateFailed
		case "process.cancelled":
			state = model.StateCancelled
		}
	}
	return state, nil
}

func (s *Service) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			return
		case id := <-s.queue:
			if s.isStopping() {
				return
			}
			s.execute(id)
		}
	}
}

func (s *Service) execute(id string) {
	ctx := s.ctx
	if s.isStopping() {
		return
	}
	s.scheduleMu.Lock()
	p, err := s.store.GetProcess(ctx, id)
	if err != nil || p.State != model.StateQueued {
		s.scheduleMu.Unlock()
		return
	}
	if !s.canAdmit(ctx, p) {
		s.scheduleMu.Unlock()
		s.requeue(id, 100*time.Millisecond)
		return
	}
	s.lifecycleMu.Lock()
	s.active.Store(id, true)
	p, err = s.store.Transition(ctx, id, model.StateRunning, "process.running", map[string]any{"attempt": p.Attempt + 1})
	if err != nil {
		s.active.Delete(id)
	}
	s.lifecycleMu.Unlock()
	s.scheduleMu.Unlock()
	if err != nil {
		return
	}
	_ = s.store.IncrementAttempt(ctx, id)
	if p.Manifest.Checkpoint.Enabled && p.Manifest.Checkpoint.ResumeOnStart {
		checkpoint, checkpointErr := s.store.LatestCheckpoint(ctx, id)
		if checkpointErr != nil {
			s.active.Delete(id)
			_, _ = s.store.TransitionWithError(context.Background(), id, model.StateFailed,
				checkpointErr.Error(), "process.failed", map[string]any{"error": checkpointErr.Error()})
			return
		}
		p.Checkpoint = checkpoint
	}
	runCtx := ctx
	var cancel context.CancelFunc
	if p.Manifest.Budget.MaxDurationSec > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(p.Manifest.Budget.MaxDurationSec)*time.Second)
		defer cancel()
	}
	emit := func(eventType string, data any) {
		_ = s.store.AppendEvent(context.Background(), id, eventType, data)
	}
	output, runErr := s.runner.Run(runCtx, p, s.handleWorkerTool, s.UpdateUsage, emit)
	s.active.Delete(id)
	if output.Usage.Tokens > 0 || output.Usage.CostUSD > 0 {
		if usageErr := s.UpdateUsage(context.Background(), id, output.Usage); usageErr != nil {
			if runErr == nil {
				runErr = usageErr
			} else {
				runErr = fmt.Errorf("%v; usage accounting: %w", runErr, usageErr)
			}
		}
	}
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.isStopping() {
		return
	}
	current, getErr := s.store.GetProcess(context.Background(), id)
	if getErr != nil || current.State != model.StateRunning {
		return
	}
	if runErr == nil {
		_, _ = s.store.Transition(context.Background(), id, model.StateSucceeded, "process.succeeded", nil)
		return
	}
	emit("process.execution_error", map[string]any{"error": runErr.Error()})
	ambiguousTool, ambiguousErr := s.store.HasStartedToolCall(context.Background(), id)
	if ambiguousErr != nil {
		runErr = fmt.Errorf("%v; inspect tool state: %w", runErr, ambiguousErr)
	}
	if current.Attempt < current.Manifest.Retry.MaxAttempts &&
		!errors.Is(runErr, context.DeadlineExceeded) &&
		!ambiguousTool {
		_, err = s.store.TransitionWithError(context.Background(), id, model.StateQueued, runErr.Error(),
			"process.retry_scheduled", map[string]any{
				"attempt": current.Attempt, "backoff_seconds": current.Manifest.Retry.BackoffSec,
			})
		if err == nil {
			s.requeue(id, time.Duration(current.Manifest.Retry.BackoffSec)*time.Second)
		}
		return
	}
	_, _ = s.store.TransitionWithError(context.Background(), id, model.StateFailed, runErr.Error(),
		"process.failed", map[string]any{"error": runErr.Error()})
}

func (s *Service) handleWorkerTool(
	ctx context.Context,
	process model.Process,
	request model.ToolRequest,
) model.ToolResult {
	decision, err := s.RequestTool(ctx, process.ID, request)
	if err != nil {
		return model.ToolResult{IdempotencyKey: request.IdempotencyKey, Status: "failed", Error: err.Error()}
	}
	for decision.Status == "waiting_approval" {
		select {
		case <-ctx.Done():
			return model.ToolResult{
				IdempotencyKey: request.IdempotencyKey, Status: "outcome_unknown", Error: ctx.Err().Error(),
			}
		case <-time.After(100 * time.Millisecond):
		}
		_, status, raw, getErr := s.store.ToolCall(ctx, process.ID, request.IdempotencyKey)
		if getErr != nil {
			return model.ToolResult{IdempotencyKey: request.IdempotencyKey, Status: "failed", Error: getErr.Error()}
		}
		decision.Status = status
		if status == "completed" || status == "failed" || status == "outcome_unknown" {
			var prior model.ToolResult
			if json.Unmarshal(raw, &prior) == nil {
				return prior
			}
		}
	}
	current, stateErr := s.store.GetProcess(ctx, process.ID)
	if stateErr != nil {
		return model.ToolResult{IdempotencyKey: request.IdempotencyKey, Status: "failed", Error: stateErr.Error()}
	}
	if current.State != model.StateRunning {
		return model.ToolResult{
			IdempotencyKey: request.IdempotencyKey,
			Status:         "failed",
			Error:          fmt.Sprintf("process is no longer running: %s", current.State),
		}
	}
	if decision.Status == "denied" {
		return model.ToolResult{IdempotencyKey: request.IdempotencyKey, Status: "failed", Error: "tool request denied"}
	}
	if decision.Status == "started" {
		result := model.ToolResult{
			IdempotencyKey: request.IdempotencyKey,
			Status:         "outcome_unknown",
			Error:          "tool execution was interrupted after it started",
		}
		_ = s.CompleteTool(context.Background(), process.ID, request.IdempotencyKey, result)
		return result
	}
	if decision.Status == "completed" || decision.Status == "failed" || decision.Status == "outcome_unknown" {
		_, _, raw, getErr := s.store.ToolCall(ctx, process.ID, request.IdempotencyKey)
		var prior model.ToolResult
		if getErr == nil && json.Unmarshal(raw, &prior) == nil {
			return prior
		}
		return model.ToolResult{IdempotencyKey: request.IdempotencyKey, Status: decision.Status}
	}
	if err = s.StartTool(ctx, process.ID, request.IdempotencyKey); err != nil {
		return model.ToolResult{IdempotencyKey: request.IdempotencyKey, Status: "failed", Error: err.Error()}
	}
	result := s.executeBrokeredTool(process, request)
	if completeErr := s.CompleteTool(context.Background(), process.ID, request.IdempotencyKey, result); completeErr != nil {
		return model.ToolResult{
			IdempotencyKey: request.IdempotencyKey,
			Status:         "outcome_unknown",
			Error:          "tool completed but durable completion failed: " + completeErr.Error(),
		}
	}
	return result
}

func (s *Service) executeBrokeredTool(process model.Process, request model.ToolRequest) model.ToolResult {
	result := model.ToolResult{IdempotencyKey: request.IdempotencyKey, Status: "completed"}
	switch request.Action {
	case "fs.write":
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			result.Status = "failed"
			result.Error = "invalid fs.write payload: " + err.Error()
			return result
		}
		hostPath, err := brokeredHostPath(process.Manifest, request.Resource)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			return result
		}
		if err = atomicWriteFile(hostPath, []byte(payload.Content)); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			return result
		}
		result.Output, _ = json.Marshal(map[string]any{"path": request.Resource, "bytes": len(payload.Content)})
	default:
		result.Status = "failed"
		result.Error = fmt.Sprintf("brokered tool %q is not implemented", request.Action)
	}
	return result
}

func brokeredHostPath(manifest model.Manifest, resource string) (string, error) {
	cleanResource := cleanContainerPath(resource)
	for _, mount := range manifest.Mounts {
		cleanTarget := cleanContainerPath(mount.Target)
		relative, ok := containerRelative(cleanTarget, cleanResource)
		if !ok {
			continue
		}
		source, err := secureBrokerRoot(mount.Source)
		if err != nil {
			return "", err
		}
		target := filepath.Join(source, filepath.FromSlash(relative))
		if err = ensureNoSymlinkTraversal(source, filepath.Dir(target)); err != nil {
			return "", err
		}
		return target, nil
	}
	return "", fmt.Errorf("resource %q is not covered by a declared mount", resource)
}

func cleanContainerPath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	parts := make([]string, 0)
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
			continue
		}
		parts = append(parts, part)
	}
	return "/" + strings.Join(parts, "/")
}

func containerRelative(root, target string) (string, bool) {
	if target == root {
		return "", true
	}
	prefix := strings.TrimSuffix(root, "/") + "/"
	if !strings.HasPrefix(target, prefix) {
		return "", false
	}
	return strings.TrimPrefix(target, prefix), true
}

func secureBrokerRoot(source string) (string, error) {
	absolute, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	workspace := os.Getenv("AGENTOS_WORKSPACE_ROOT")
	if workspace == "" {
		workspace, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	workspace, err = filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(workspace, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("brokered path is outside AGENTOS_WORKSPACE_ROOT")
	}
	if err = ensureNoSymlinkTraversal(workspace, absolute); err != nil {
		return "", err
	}
	return absolute, nil
}

func ensureNoSymlinkTraversal(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("path escapes its declared root")
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path traverses symbolic link %q", current)
		}
	}
	return nil
}

func atomicWriteFile(target string, content []byte) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(parent, ".agentos-write-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err = temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tempName, 0o644); err != nil {
		return err
	}
	return os.Rename(tempName, target)
}

func (s *Service) enqueue(id string) {
	if s.isStopping() {
		return
	}
	select {
	case s.queue <- id:
	default:
		s.requeue(id, 10*time.Millisecond)
	}
}

func (s *Service) requeue(id string, delay time.Duration) {
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-s.ctx.Done():
			return
		case <-timer.C:
		}
		select {
		case <-s.ctx.Done():
		case s.queue <- id:
		}
	}()
}

func (s *Service) isStopping() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.stopping
}

func (s *Service) canAdmit(ctx context.Context, process model.Process) bool {
	ancestors, err := s.store.Ancestors(ctx, process.ID)
	if err != nil {
		return false
	}
	for _, ancestor := range ancestors {
		if ancestor.Manifest.Budget.MaxConcurrency < 1 {
			continue
		}
		running, countErr := s.store.RunningDescendantCount(ctx, ancestor.ID)
		if countErr != nil || running >= ancestor.Manifest.Budget.MaxConcurrency {
			return false
		}
	}
	return true
}

func (s *Service) validateChild(ctx context.Context, child model.Manifest) error {
	parent, err := s.store.GetProcess(ctx, child.ParentID)
	if err != nil {
		return fmt.Errorf("parent process: %w", err)
	}
	if parent.State.Terminal() {
		return errors.New("terminal parent cannot create children")
	}
	count, err := s.store.CountChildren(ctx, child.ParentID)
	if err != nil {
		return err
	}
	if parent.Manifest.Budget.MaxChildren == 0 {
		return errors.New("parent does not allow child processes")
	}
	if count >= parent.Manifest.Budget.MaxChildren {
		return errors.New("parent child limit exceeded")
	}
	if !budgetWithin(child.Budget, parent.Manifest.Budget) {
		return errors.New("child budget exceeds parent budget")
	}
	if !capabilitiesSubset(child.Capabilities, parent.Manifest.Capabilities) {
		return errors.New("child capabilities must be a subset of parent capabilities")
	}
	if !approvalRulesPreserved(child.ApprovalRules, parent.Manifest.ApprovalRules) {
		return errors.New("child approval rules must preserve all parent approval requirements")
	}
	if child.Image != parent.Manifest.Image ||
		child.Implementation.Adapter != parent.Manifest.Implementation.Adapter ||
		!stringSliceEqual(child.Implementation.Command, parent.Manifest.Implementation.Command) ||
		!envSubset(child.Implementation.Env, parent.Manifest.Implementation.Env) ||
		!mountsSubset(child.Mounts, parent.Manifest.Mounts) {
		return errors.New("child runtime image, command, environment, and mounts must not widen the parent")
	}
	reserved, err := s.store.ChildReservations(ctx, child.ParentID)
	if err != nil {
		return err
	}
	if parent.Manifest.Budget.MaxTokens > 0 && reserved.MaxTokens+child.Budget.MaxTokens > parent.Manifest.Budget.MaxTokens {
		return errors.New("child token reservation exceeds remaining parent budget")
	}
	if parent.Manifest.Budget.MaxCostUSD > 0 && reserved.MaxCostUSD+child.Budget.MaxCostUSD > parent.Manifest.Budget.MaxCostUSD {
		return errors.New("child cost reservation exceeds remaining parent budget")
	}
	if parent.Manifest.Budget.MaxConcurrency > 0 && reserved.MaxConcurrency+child.Budget.MaxConcurrency > parent.Manifest.Budget.MaxConcurrency {
		return errors.New("child concurrency reservation exceeds parent budget")
	}
	if parent.Manifest.Budget.MaxDurationSec > 0 &&
		reserved.MaxDurationSec+child.Budget.MaxDurationSec > parent.Manifest.Budget.MaxDurationSec {
		return errors.New("child duration reservation exceeds parent budget")
	}
	return nil
}

func budgetWithin(child, parent model.Budget) bool {
	if parent.MaxTokens > 0 && (child.MaxTokens == 0 || child.MaxTokens > parent.MaxTokens) {
		return false
	}
	if parent.MaxCostUSD > 0 && (child.MaxCostUSD == 0 || child.MaxCostUSD > parent.MaxCostUSD) {
		return false
	}
	if parent.MaxDurationSec > 0 && (child.MaxDurationSec == 0 || child.MaxDurationSec > parent.MaxDurationSec) {
		return false
	}
	if child.MaxChildren > parent.MaxChildren {
		return false
	}
	if child.MaxConcurrency > parent.MaxConcurrency {
		return false
	}
	return true
}

func capabilitiesSubset(child, parent model.Capabilities) bool {
	return stringSubset(child.Tools, parent.Tools) &&
		stringSubset(child.FilesystemRead, parent.FilesystemRead) &&
		stringSubset(child.FilesystemWrite, parent.FilesystemWrite) &&
		stringSubset(child.NetworkDestinations, parent.NetworkDestinations) &&
		stringSubset(child.Secrets, parent.Secrets)
}

func approvalRulesPreserved(child, parent []model.ApprovalRule) bool {
	for _, required := range parent {
		found := false
		for _, candidate := range child {
			if candidate.Action == required.Action && candidate.Match == required.Match {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func stringSubset(child, parent []string) bool {
	set := make(map[string]bool, len(parent))
	for _, value := range parent {
		set[value] = true
	}
	for _, value := range child {
		if !set[value] {
			return false
		}
	}
	return true
}

func stringSliceEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func envSubset(child, parent map[string]string) bool {
	for key, value := range child {
		if parent[key] != value {
			return false
		}
	}
	return true
}

func mountsSubset(child, parent []model.Mount) bool {
	for _, candidate := range child {
		found := false
		for _, allowed := range parent {
			if candidate.Source == allowed.Source && candidate.Target == allowed.Target &&
				(candidate.ReadOnly || !allowed.ReadOnly) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
