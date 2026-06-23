package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ProcessState string

const (
	StateCreated         ProcessState = "created"
	StateQueued          ProcessState = "queued"
	StateRunning         ProcessState = "running"
	StateWaitingApproval ProcessState = "waiting_approval"
	StateSuspended       ProcessState = "suspended"
	StateSucceeded       ProcessState = "succeeded"
	StateFailed          ProcessState = "failed"
	StateCancelled       ProcessState = "cancelled"
)

var terminalStates = map[ProcessState]bool{
	StateSucceeded: true,
	StateFailed:    true,
	StateCancelled: true,
}

func (s ProcessState) Terminal() bool { return terminalStates[s] }

var transitions = map[ProcessState]map[ProcessState]bool{
	StateCreated:         {StateQueued: true, StateCancelled: true},
	StateQueued:          {StateRunning: true, StateSuspended: true, StateCancelled: true, StateFailed: true},
	StateRunning:         {StateQueued: true, StateWaitingApproval: true, StateSuspended: true, StateSucceeded: true, StateFailed: true, StateCancelled: true},
	StateWaitingApproval: {StateRunning: true, StateSuspended: true, StateFailed: true, StateCancelled: true},
	StateSuspended:       {StateQueued: true, StateCancelled: true},
}

func CanTransition(from, to ProcessState) bool {
	return transitions[from][to]
}

type Budget struct {
	MaxTokens      int64   `json:"max_tokens" yaml:"max_tokens"`
	MaxCostUSD     float64 `json:"max_cost_usd" yaml:"max_cost_usd"`
	MaxDurationSec int64   `json:"max_duration_seconds" yaml:"max_duration_seconds"`
	MaxConcurrency int     `json:"max_concurrency" yaml:"max_concurrency"`
	MaxChildren    int     `json:"max_children" yaml:"max_children"`
}

type Usage struct {
	Tokens  int64   `json:"tokens"`
	CostUSD float64 `json:"cost_usd"`
}

type ModelPricing struct {
	InputUSDPerMillion  float64 `json:"input_usd_per_million" yaml:"input_usd_per_million"`
	OutputUSDPerMillion float64 `json:"output_usd_per_million" yaml:"output_usd_per_million"`
}

type Mount struct {
	Source   string `json:"source" yaml:"source"`
	Target   string `json:"target" yaml:"target"`
	ReadOnly bool   `json:"read_only" yaml:"read_only"`
}

type Capabilities struct {
	Tools               []string `json:"tools" yaml:"tools"`
	FilesystemRead      []string `json:"filesystem_read" yaml:"filesystem_read"`
	FilesystemWrite     []string `json:"filesystem_write" yaml:"filesystem_write"`
	NetworkDestinations []string `json:"network_destinations" yaml:"network_destinations"`
	Secrets             []string `json:"secrets" yaml:"secrets"`
}

type ApprovalRule struct {
	Action string `json:"action" yaml:"action"`
	Match  string `json:"match" yaml:"match"`
}

type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts" yaml:"max_attempts"`
	BackoffSec  int `json:"backoff_seconds" yaml:"backoff_seconds"`
}

type CheckpointPolicy struct {
	Enabled       bool `json:"enabled" yaml:"enabled"`
	IntervalSec   int  `json:"interval_seconds" yaml:"interval_seconds"`
	ResumeOnStart bool `json:"resume_on_start" yaml:"resume_on_start"`
}

type Implementation struct {
	Adapter string            `json:"adapter" yaml:"adapter"`
	Command []string          `json:"command" yaml:"command"`
	Env     map[string]string `json:"env" yaml:"env"`
}

type Manifest struct {
	Name           string           `json:"name" yaml:"name"`
	Image          string           `json:"image" yaml:"image"`
	Task           string           `json:"task" yaml:"task"`
	Model          string           `json:"model" yaml:"model"`
	Pricing        ModelPricing     `json:"pricing" yaml:"pricing"`
	Implementation Implementation   `json:"implementation" yaml:"implementation"`
	Mounts         []Mount          `json:"mounts" yaml:"mounts"`
	Capabilities   Capabilities     `json:"capabilities" yaml:"capabilities"`
	Budget         Budget           `json:"budget" yaml:"budget"`
	ApprovalRules  []ApprovalRule   `json:"approval_rules" yaml:"approval_rules"`
	Retry          RetryPolicy      `json:"retry" yaml:"retry"`
	Checkpoint     CheckpointPolicy `json:"checkpoint" yaml:"checkpoint"`
	ParentID       string           `json:"parent_id,omitempty" yaml:"parent_id,omitempty"`
}

func (m *Manifest) ApplyDefaults() {
	if m.Name == "" {
		m.Name = "agent"
	}
	if m.Budget.MaxConcurrency == 0 {
		m.Budget.MaxConcurrency = 1
	}
	if m.Retry.MaxAttempts == 0 {
		m.Retry.MaxAttempts = 1
	}
	if m.Implementation.Adapter == "" {
		m.Implementation.Adapter = "process"
	}
}

func (m Manifest) Validate() error {
	if strings.TrimSpace(m.Image) == "" {
		return errors.New("manifest image is required")
	}
	if strings.TrimSpace(m.Task) == "" {
		return errors.New("manifest task is required")
	}
	if len(m.Implementation.Command) == 0 {
		return errors.New("manifest implementation.command is required")
	}
	if m.Budget.MaxConcurrency < 1 {
		return errors.New("budget.max_concurrency must be at least 1")
	}
	if m.Budget.MaxTokens < 0 || m.Budget.MaxCostUSD < 0 || m.Budget.MaxDurationSec < 0 || m.Budget.MaxChildren < 0 {
		return errors.New("budget values cannot be negative")
	}
	if m.Retry.MaxAttempts < 1 || m.Retry.BackoffSec < 0 {
		return errors.New("retry.max_attempts must be at least 1 and backoff_seconds cannot be negative")
	}
	if m.Checkpoint.IntervalSec < 0 {
		return errors.New("checkpoint.interval_seconds cannot be negative")
	}
	if m.Checkpoint.ResumeOnStart && !m.Checkpoint.Enabled {
		return errors.New("checkpoint.resume_on_start requires checkpoint.enabled")
	}
	if m.Pricing.InputUSDPerMillion < 0 || m.Pricing.OutputUSDPerMillion < 0 {
		return errors.New("model pricing values cannot be negative")
	}
	if m.Budget.MaxCostUSD > 0 && m.Pricing.InputUSDPerMillion == 0 && m.Pricing.OutputUSDPerMillion == 0 {
		return errors.New("a cost budget requires explicit model pricing")
	}
	for _, mount := range m.Mounts {
		if mount.Source == "" || mount.Target == "" {
			return errors.New("mount source and target are required")
		}
		if !strings.HasPrefix(mount.Target, "/") {
			return fmt.Errorf("mount target %q must be an absolute container path", mount.Target)
		}
		if !mount.ReadOnly && approvalOverlaps(m.ApprovalRules, "fs.write", mount.Target) {
			return fmt.Errorf(
				"writable mount %q bypasses its fs.write approval rule; use a read-only mount and the brokered fs.write tool",
				mount.Target,
			)
		}
	}
	for _, destination := range m.Capabilities.NetworkDestinations {
		if destination == "" ||
			destination != strings.ToLower(destination) ||
			destination != strings.TrimSuffix(destination, ".") ||
			strings.ContainsAny(destination, "/:*@") {
			return fmt.Errorf("network destination %q must be a lowercase DNS hostname without scheme, port, path, or wildcard", destination)
		}
		if approvalMatches(m.ApprovalRules, "network.connect", destination) {
			return fmt.Errorf(
				"direct network destination %q bypasses its approval rule; use a brokered network tool",
				destination,
			)
		}
	}
	for _, secret := range m.Capabilities.Secrets {
		if approvalMatches(m.ApprovalRules, "secret.read", secret) {
			return fmt.Errorf(
				"direct secret %q bypasses its approval rule; use a brokered secret tool",
				secret,
			)
		}
	}
	return nil
}

func approvalMatches(rules []ApprovalRule, action, resource string) bool {
	for _, rule := range rules {
		if rule.Action != action {
			continue
		}
		if rule.Match == "" || strings.Contains(resource, rule.Match) {
			return true
		}
	}
	return false
}

func approvalOverlaps(rules []ApprovalRule, action, root string) bool {
	cleanRoot := strings.TrimSuffix(strings.ReplaceAll(root, "\\", "/"), "/")
	for _, rule := range rules {
		if rule.Action != action {
			continue
		}
		if rule.Match == "" {
			return true
		}
		cleanMatch := strings.TrimSuffix(strings.ReplaceAll(rule.Match, "\\", "/"), "/")
		if cleanMatch == cleanRoot ||
			strings.HasPrefix(cleanMatch, cleanRoot+"/") ||
			strings.HasPrefix(cleanRoot, cleanMatch+"/") {
			return true
		}
	}
	return false
}

type Process struct {
	ID         string          `json:"id"`
	ParentID   string          `json:"parent_id,omitempty"`
	Name       string          `json:"name"`
	State      ProcessState    `json:"state"`
	Manifest   Manifest        `json:"manifest"`
	Usage      Usage           `json:"usage"`
	Attempt    int             `json:"attempt"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	Checkpoint json.RawMessage `json:"-"`
}

type Event struct {
	Sequence  int64           `json:"sequence"`
	ProcessID string          `json:"process_id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

type Approval struct {
	ID             string          `json:"id"`
	ProcessID      string          `json:"process_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Action         string          `json:"action"`
	Resource       string          `json:"resource"`
	ActionDigest   string          `json:"action_digest"`
	Payload        json.RawMessage `json:"payload"`
	Status         string          `json:"status"`
	DecisionReason string          `json:"decision_reason,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	DecidedAt      *time.Time      `json:"decided_at,omitempty"`
}

type ToolRequest struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Action         string          `json:"action"`
	Resource       string          `json:"resource"`
	Payload        json.RawMessage `json:"payload"`
}

type ToolResult struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Status         string          `json:"status"`
	Output         json.RawMessage `json:"output,omitempty"`
	Error          string          `json:"error,omitempty"`
}

type ToolCall struct {
	ProcessID      string          `json:"process_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Action         string          `json:"action"`
	RequestHash    string          `json:"request_hash"`
	Request        json.RawMessage `json:"request"`
	Status         string          `json:"status"`
	Result         json.RawMessage `json:"result,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type ActionEnvelope struct {
	ProcessID      string          `json:"process_id"`
	IdempotencyKey string          `json:"idempotency_key"`
	Action         string          `json:"action"`
	Resource       string          `json:"resource"`
	Payload        json.RawMessage `json:"payload"`
	Image          string          `json:"image"`
	PolicyVersion  string          `json:"policy_version"`
}

type ActionDecision struct {
	Status     string `json:"status"`
	ApprovalID string `json:"approval_id,omitempty"`
	Digest     string `json:"digest"`
	Reason     string `json:"reason,omitempty"`
}
