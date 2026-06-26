package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/agentos/agentos/internal/model"
	"github.com/agentos/agentos/internal/state"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := state.EnsureDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if err := state.EnsureFile(candidate); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;

CREATE TABLE IF NOT EXISTS processes (
	id TEXT PRIMARY KEY,
	parent_id TEXT REFERENCES processes(id),
	name TEXT NOT NULL,
	state TEXT NOT NULL,
	manifest_json BLOB NOT NULL,
	usage_json BLOB NOT NULL,
	attempt INTEGER NOT NULL DEFAULT 0,
	error TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_processes_state ON processes(state);
CREATE INDEX IF NOT EXISTS idx_processes_parent ON processes(parent_id);

CREATE TABLE IF NOT EXISTS events (
	sequence INTEGER PRIMARY KEY AUTOINCREMENT,
	process_id TEXT NOT NULL REFERENCES processes(id),
	type TEXT NOT NULL,
	data_json BLOB NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_process_sequence ON events(process_id, sequence);

CREATE TABLE IF NOT EXISTS approvals (
	id TEXT PRIMARY KEY,
	process_id TEXT NOT NULL REFERENCES processes(id),
	idempotency_key TEXT NOT NULL,
	action TEXT NOT NULL,
	resource TEXT NOT NULL DEFAULT '',
	action_digest TEXT NOT NULL,
	policy_version TEXT NOT NULL,
	payload_json BLOB NOT NULL,
	status TEXT NOT NULL,
	decision_reason TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	decided_at TEXT,
	UNIQUE(process_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS tool_calls (
	process_id TEXT NOT NULL REFERENCES processes(id),
	idempotency_key TEXT NOT NULL,
	action TEXT NOT NULL,
	request_hash TEXT NOT NULL,
	request_json BLOB NOT NULL,
	status TEXT NOT NULL,
	result_json BLOB,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY(process_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS customers (
	id TEXT PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	stripe_customer_id TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_customers_stripe_id ON customers(stripe_customer_id);

CREATE TABLE IF NOT EXISTS subscriptions (
	stripe_subscription_id TEXT PRIMARY KEY,
	customer_id TEXT NOT NULL REFERENCES customers(id),
	plan TEXT NOT NULL,
	status TEXT NOT NULL,
	current_period_end TEXT,
	cancel_at_period_end INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer ON subscriptions(customer_id);

CREATE TABLE IF NOT EXISTS billing_events (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	payload_hash TEXT NOT NULL,
	status TEXT NOT NULL,
	processed_at TEXT NOT NULL
);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE approvals ADD COLUMN resource TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	return nil
}

func (s *Store) CreateProcess(ctx context.Context, p model.Process) error {
	manifest, err := json.Marshal(p.Manifest)
	if err != nil {
		return err
	}
	usage, err := json.Marshal(p.Usage)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO processes(id,parent_id,name,state,manifest_json,usage_json,attempt,error,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		p.ID, nullable(p.ParentID), p.Name, p.State, manifest, usage, p.Attempt, p.Error,
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt)); err != nil {
		return err
	}
	if err = appendEventTx(ctx, tx, p.ID, "process.created", map[string]any{"state": p.State}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetProcess(ctx context.Context, id string) (model.Process, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id,COALESCE(parent_id,''),name,state,manifest_json,usage_json,attempt,error,created_at,updated_at
		FROM processes WHERE id=?`, id)
	return scanProcess(row)
}

func (s *Store) ListProcesses(ctx context.Context) ([]model.Process, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,COALESCE(parent_id,''),name,state,manifest_json,usage_json,attempt,error,created_at,updated_at
		FROM processes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Process, 0)
	for rows.Next() {
		p, err := scanProcess(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProcess(row scanner) (model.Process, error) {
	var p model.Process
	var manifest, usage []byte
	var created, updated string
	err := row.Scan(&p.ID, &p.ParentID, &p.Name, &p.State, &manifest, &usage, &p.Attempt, &p.Error, &created, &updated)
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal(manifest, &p.Manifest); err != nil {
		return p, err
	}
	if err = json.Unmarshal(usage, &p.Usage); err != nil {
		return p, err
	}
	p.CreatedAt, err = parseTime(created)
	if err != nil {
		return p, err
	}
	p.UpdatedAt, err = parseTime(updated)
	return p, err
}

func (s *Store) Transition(ctx context.Context, id string, to model.ProcessState, eventType string, data any) (model.Process, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Process{}, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
		SELECT id,COALESCE(parent_id,''),name,state,manifest_json,usage_json,attempt,error,created_at,updated_at
		FROM processes WHERE id=?`, id)
	p, err := scanProcess(row)
	if err != nil {
		return p, err
	}
	if !model.CanTransition(p.State, to) {
		return p, fmt.Errorf("invalid process transition %s -> %s", p.State, to)
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE processes SET state=?,updated_at=? WHERE id=? AND state=?`,
		to, formatTime(now), id, p.State)
	if err != nil {
		return p, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return p, err
	}
	if n != 1 {
		return p, errors.New("process state changed concurrently")
	}
	if data == nil {
		data = map[string]any{}
	}
	if err = appendEventTx(ctx, tx, id, eventType, data); err != nil {
		return p, err
	}
	if err = tx.Commit(); err != nil {
		return p, err
	}
	p.State = to
	p.UpdatedAt = now
	return p, nil
}

func (s *Store) SetFailure(ctx context.Context, id, message string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE processes SET error=?,updated_at=? WHERE id=?`,
		message, formatTime(time.Now().UTC()), id)
	return err
}

func (s *Store) TransitionWithError(
	ctx context.Context,
	id string,
	to model.ProcessState,
	message, eventType string,
	data any,
) (model.Process, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Process{}, err
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `
		SELECT id,COALESCE(parent_id,''),name,state,manifest_json,usage_json,attempt,error,created_at,updated_at
		FROM processes WHERE id=?`, id)
	p, err := scanProcess(row)
	if err != nil {
		return p, err
	}
	if !model.CanTransition(p.State, to) {
		return p, fmt.Errorf("invalid process transition %s -> %s", p.State, to)
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `
		UPDATE processes SET state=?,error=?,updated_at=? WHERE id=? AND state=?`,
		to, message, formatTime(now), id, p.State)
	if err != nil {
		return p, err
	}
	if count, rowsErr := result.RowsAffected(); rowsErr != nil || count != 1 {
		if rowsErr != nil {
			return p, rowsErr
		}
		return p, errors.New("process state changed concurrently")
	}
	if data == nil {
		data = map[string]any{}
	}
	if err = appendEventTx(ctx, tx, id, eventType, data); err != nil {
		return p, err
	}
	p.State = to
	p.Error = message
	p.UpdatedAt = now
	return p, tx.Commit()
}

func (s *Store) IncrementAttempt(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE processes SET attempt=attempt+1,updated_at=? WHERE id=?`,
		formatTime(time.Now().UTC()), id)
	return err
}

func (s *Store) UpdateUsage(ctx context.Context, id string, usage model.Usage) error {
	raw, err := json.Marshal(usage)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE processes SET usage_json=?,updated_at=? WHERE id=?`,
		raw, formatTime(time.Now().UTC()), id); err != nil {
		return err
	}
	if err = appendEventTx(ctx, tx, id, "budget.usage_updated", usage); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Events(ctx context.Context, id string) ([]model.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT sequence,process_id,type,data_json,created_at
		FROM events WHERE process_id=? ORDER BY sequence`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Event, 0)
	for rows.Next() {
		var e model.Event
		var created string
		if err = rows.Scan(&e.Sequence, &e.ProcessID, &e.Type, &e.Data, &created); err != nil {
			return nil, err
		}
		e.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *Store) LatestCheckpoint(ctx context.Context, processID string) (json.RawMessage, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `
		SELECT data_json FROM events
		WHERE process_id=? AND type='process.checkpoint'
		ORDER BY sequence DESC LIMIT 1`, processID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return json.RawMessage(raw), err
}

func (s *Store) AppendEvent(ctx context.Context, processID, eventType string, data any) error {
	return appendEventTx(ctx, s.db, processID, eventType, data)
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func appendEventTx(ctx context.Context, exec execer, processID, eventType string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = exec.ExecContext(ctx, `INSERT INTO events(process_id,type,data_json,created_at) VALUES(?,?,?,?)`,
		processID, eventType, raw, formatTime(time.Now().UTC()))
	return err
}

func (s *Store) Recover(ctx context.Context) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `
		SELECT id,state FROM processes
		WHERE state IN ('queued','running','waiting_approval')`)
	if err != nil {
		return nil, err
	}
	var ids []string
	var states []model.ProcessState
	for rows.Next() {
		var id string
		var state model.ProcessState
		if err = rows.Scan(&id, &state); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
		states = append(states, state)
	}
	rows.Close()
	for i, id := range ids {
		target := model.StateQueued
		event := "process.recovered"
		if states[i] == model.StateWaitingApproval {
			target = model.StateWaitingApproval
		}
		if target != states[i] {
			if _, err = tx.ExecContext(ctx, `UPDATE processes SET state=?,updated_at=? WHERE id=?`,
				target, formatTime(time.Now().UTC()), id); err != nil {
				return nil, err
			}
		}
		if err = appendEventTx(ctx, tx, id, event, map[string]any{"from": states[i], "to": target}); err != nil {
			return nil, err
		}
	}
	return ids, tx.Commit()
}

func (s *Store) RunningDescendantCount(ctx context.Context, rootID string) (int, error) {
	const query = `
WITH RECURSIVE descendants(id) AS (
	SELECT id FROM processes WHERE id=?
	UNION ALL
	SELECT p.id FROM processes p JOIN descendants d ON p.parent_id=d.id
)
SELECT COUNT(*) FROM processes WHERE id IN descendants AND state='running'`
	var count int
	err := s.db.QueryRowContext(ctx, query, rootID).Scan(&count)
	return count, err
}

func (s *Store) CountChildren(ctx context.Context, parentID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM processes WHERE parent_id=?`, parentID).Scan(&count)
	return count, err
}

func (s *Store) DescendantUsage(ctx context.Context, rootID string) (model.Usage, error) {
	const query = `
WITH RECURSIVE descendants(id) AS (
	SELECT id FROM processes WHERE id=?
	UNION ALL
	SELECT p.id FROM processes p JOIN descendants d ON p.parent_id=d.id
)
SELECT COALESCE(SUM(CAST(json_extract(usage_json,'$.tokens') AS INTEGER)),0),
       COALESCE(SUM(CAST(json_extract(usage_json,'$.cost_usd') AS REAL)),0)
FROM processes WHERE id IN descendants`
	var usage model.Usage
	err := s.db.QueryRowContext(ctx, query, rootID).Scan(&usage.Tokens, &usage.CostUSD)
	return usage, err
}

func (s *Store) ChildReservations(ctx context.Context, parentID string) (model.Budget, error) {
	const query = `
SELECT COALESCE(SUM(CAST(json_extract(manifest_json,'$.budget.max_tokens') AS INTEGER)),0),
       COALESCE(SUM(CAST(json_extract(manifest_json,'$.budget.max_cost_usd') AS REAL)),0),
       COALESCE(SUM(CAST(json_extract(manifest_json,'$.budget.max_concurrency') AS INTEGER)),0),
       COALESCE(SUM(CAST(json_extract(manifest_json,'$.budget.max_duration_seconds') AS INTEGER)),0)
FROM processes WHERE parent_id=?`
	var budget model.Budget
	err := s.db.QueryRowContext(ctx, query, parentID).
		Scan(&budget.MaxTokens, &budget.MaxCostUSD, &budget.MaxConcurrency, &budget.MaxDurationSec)
	return budget, err
}

func (s *Store) Ancestors(ctx context.Context, processID string) ([]model.Process, error) {
	const query = `
WITH RECURSIVE ancestors(id,parent_id,depth) AS (
	SELECT id,parent_id,0 FROM processes WHERE id=?
	UNION ALL
	SELECT p.id,p.parent_id,a.depth+1
	FROM processes p JOIN ancestors a ON a.parent_id=p.id
)
SELECT p.id,COALESCE(p.parent_id,''),p.name,p.state,p.manifest_json,p.usage_json,p.attempt,p.error,p.created_at,p.updated_at
FROM processes p JOIN ancestors a ON p.id=a.id ORDER BY a.depth`
	rows, err := s.db.QueryContext(ctx, query, processID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.Process
	for rows.Next() {
		p, scanErr := scanProcess(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *Store) CreateToolCall(ctx context.Context, processID, key, action, requestHash string, request any) (bool, error) {
	raw, err := json.Marshal(request)
	if err != nil {
		return false, err
	}
	now := formatTime(time.Now().UTC())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO tool_calls(process_id,idempotency_key,action,request_hash,request_json,status,created_at,updated_at)
		VALUES(?,?,?,?,?,'requested',?,?)`,
		processID, key, action, requestHash, raw, now, now)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return false, err
	}
	if err = appendEventTx(ctx, tx, processID, "tool.requested", map[string]any{
		"idempotency_key": key, "action": action, "digest": requestHash,
	}); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (s *Store) ToolCall(ctx context.Context, processID, key string) (requestHash, status string, result json.RawMessage, err error) {
	var raw string
	err = s.db.QueryRowContext(ctx, `
		SELECT request_hash,status,COALESCE(result_json,'null')
		FROM tool_calls WHERE process_id=? AND idempotency_key=?`, processID, key).
		Scan(&requestHash, &status, &raw)
	result = json.RawMessage(raw)
	return
}

func (s *Store) HasStartedToolCall(ctx context.Context, processID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tool_calls WHERE process_id=? AND status='started'`, processID).Scan(&count)
	return count > 0, err
}

func (s *Store) UpdateToolCall(ctx context.Context, processID, key, status string, result any) error {
	var raw []byte
	var err error
	if result != nil {
		raw, err = json.Marshal(result)
		if err != nil {
			return err
		}
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE tool_calls SET status=?,result_json=?,updated_at=?
		WHERE process_id=? AND idempotency_key=?`,
		status, raw, formatTime(time.Now().UTC()), processID, key)
	return err
}

func (s *Store) UpdateToolCallWithEvent(
	ctx context.Context,
	processID, key, status string,
	result any,
	eventType string,
	eventData any,
) error {
	var raw []byte
	var err error
	if result != nil {
		raw, err = json.Marshal(result)
		if err != nil {
			return err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	update, err := tx.ExecContext(ctx, `
		UPDATE tool_calls SET status=?,result_json=?,updated_at=?
		WHERE process_id=? AND idempotency_key=?`,
		status, raw, formatTime(time.Now().UTC()), processID, key)
	if err != nil {
		return err
	}
	if count, rowsErr := update.RowsAffected(); rowsErr != nil || count != 1 {
		if rowsErr != nil {
			return rowsErr
		}
		return sql.ErrNoRows
	}
	if err = appendEventTx(ctx, tx, processID, eventType, eventData); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ClaimToolCall(ctx context.Context, processID, key string, allowed []string, target string) (bool, error) {
	if len(allowed) == 0 {
		return false, errors.New("allowed tool states are required")
	}
	query := `UPDATE tool_calls SET status=?,updated_at=? WHERE process_id=? AND idempotency_key=? AND status IN (`
	args := []any{target, formatTime(time.Now().UTC()), processID, key}
	for i, state := range allowed {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, state)
	}
	query += ")"
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

func (s *Store) ClaimToolCallWithEvent(ctx context.Context, processID, key string) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	update, err := tx.ExecContext(ctx, `
		UPDATE tool_calls SET status='started',updated_at=?
		WHERE process_id=? AND idempotency_key=? AND status='authorized'`,
		formatTime(time.Now().UTC()), processID, key)
	if err != nil {
		return false, err
	}
	n, err := update.RowsAffected()
	if err != nil || n != 1 {
		return false, err
	}
	if err = appendEventTx(ctx, tx, processID, "tool.started", map[string]any{"idempotency_key": key}); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (s *Store) CompleteToolCall(ctx context.Context, processID, key, target string, result any) (bool, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	update, err := tx.ExecContext(ctx, `
		UPDATE tool_calls SET status=?,result_json=?,updated_at=?
		WHERE process_id=? AND idempotency_key=? AND status='started'`,
		target, raw, formatTime(time.Now().UTC()), processID, key)
	if err != nil {
		return false, err
	}
	n, err := update.RowsAffected()
	if err != nil || n != 1 {
		return false, err
	}
	if err = appendEventTx(ctx, tx, processID, "tool."+target, map[string]any{
		"idempotency_key": key, "status": target,
	}); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (s *Store) CreateApproval(ctx context.Context, a model.Approval, actionDigest, policyVersion string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO approvals(id,process_id,idempotency_key,action,resource,action_digest,policy_version,payload_json,status,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.ProcessID, a.IdempotencyKey, a.Action, a.Resource, actionDigest, policyVersion, a.Payload,
		a.Status, formatTime(a.CreatedAt)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE tool_calls SET status='waiting_approval',updated_at=?
		WHERE process_id=? AND idempotency_key=?`,
		formatTime(time.Now().UTC()), a.ProcessID, a.IdempotencyKey); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE processes SET state='waiting_approval',updated_at=?
		WHERE id=? AND state='running'`,
		formatTime(time.Now().UTC()), a.ProcessID)
	if err != nil {
		return err
	}
	if changed, rowsErr := result.RowsAffected(); rowsErr != nil {
		return rowsErr
	} else if changed == 1 {
		if err = appendEventTx(ctx, tx, a.ProcessID, "process.waiting_approval",
			map[string]any{"approval_id": a.ID}); err != nil {
			return err
		}
	}
	if err = appendEventTx(ctx, tx, a.ProcessID, "approval.requested", map[string]any{
		"approval_id": a.ID, "digest": actionDigest, "action": a.Action,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetApproval(ctx context.Context, id string) (model.Approval, string, string, error) {
	var a model.Approval
	var digest, policyVersion, created string
	var decided sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id,process_id,idempotency_key,action,resource,payload_json,status,decision_reason,
		       action_digest,policy_version,created_at,decided_at
		FROM approvals WHERE id=?`, id).
		Scan(&a.ID, &a.ProcessID, &a.IdempotencyKey, &a.Action, &a.Resource, &a.Payload, &a.Status,
			&a.DecisionReason, &digest, &policyVersion, &created, &decided)
	if err != nil {
		return a, "", "", err
	}
	a.CreatedAt, err = parseTime(created)
	a.ActionDigest = digest
	if err == nil && decided.Valid {
		var t time.Time
		t, err = parseTime(decided.String)
		a.DecidedAt = &t
	}
	return a, digest, policyVersion, err
}

func (s *Store) DecideApproval(ctx context.Context, id, expectedStatus, decision, reason string) (bool, error) {
	now := formatTime(time.Now().UTC())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var processID, key, digest string
	if err = tx.QueryRowContext(ctx, `
		SELECT process_id,idempotency_key,action_digest FROM approvals WHERE id=?`, id).
		Scan(&processID, &key, &digest); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE approvals SET status=?,decision_reason=?,decided_at=?
		WHERE id=? AND status=?`, decision, reason, now, id, expectedStatus)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return false, err
	}
	toolStatus := "authorized"
	if decision == "denied" {
		toolStatus = "denied"
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE tool_calls SET status=?,result_json=?,updated_at=?
		WHERE process_id=? AND idempotency_key=?`,
		toolStatus, []byte(fmt.Sprintf(`{"approval_id":%q}`, id)), now, processID, key); err != nil {
		return false, err
	}
	if err = appendEventTx(ctx, tx, processID, "approval."+decision,
		map[string]any{"approval_id": id, "digest": digest}); err != nil {
		return false, err
	}
	target := model.StateRunning
	eventType := "process.approval_resumed"
	data := map[string]any{"approval_id": id}
	if decision == "denied" {
		target = model.StateFailed
		eventType = "process.failed"
		data["reason"] = "approval denied"
	}
	stateResult, err := tx.ExecContext(ctx, `
		UPDATE processes SET state=?,error=CASE WHEN ?='failed' THEN 'required action was denied' ELSE error END,updated_at=?
		WHERE id=? AND state='waiting_approval'`,
		target, target, now, processID)
	if err != nil {
		return false, err
	}
	if changed, rowsErr := stateResult.RowsAffected(); rowsErr != nil {
		return false, rowsErr
	} else if changed == 1 {
		if err = appendEventTx(ctx, tx, processID, eventType, data); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}

func (s *Store) PendingApprovals(ctx context.Context) ([]model.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,process_id,idempotency_key,action,resource,action_digest,payload_json,status,decision_reason,created_at
		FROM approvals WHERE status='pending' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Approval, 0)
	for rows.Next() {
		var a model.Approval
		var created string
		if err = rows.Scan(&a.ID, &a.ProcessID, &a.IdempotencyKey, &a.Action, &a.Resource, &a.ActionDigest, &a.Payload,
			&a.Status, &a.DecisionReason, &created); err != nil {
			return nil, err
		}
		a.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *Store) ApprovalForTool(ctx context.Context, processID, key string) (model.Approval, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM approvals WHERE process_id=? AND idempotency_key=?`, processID, key).Scan(&id)
	if err != nil {
		return model.Approval{}, err
	}
	a, _, _, err := s.GetApproval(ctx, id)
	return a, err
}

func (s *Store) ApprovalsForProcess(ctx context.Context, processID string) ([]model.Approval, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,process_id,idempotency_key,action,resource,action_digest,payload_json,status,decision_reason,created_at,decided_at
		FROM approvals WHERE process_id=? ORDER BY created_at,id`, processID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.Approval, 0)
	for rows.Next() {
		var a model.Approval
		var created string
		var decided sql.NullString
		if err = rows.Scan(&a.ID, &a.ProcessID, &a.IdempotencyKey, &a.Action, &a.Resource, &a.ActionDigest, &a.Payload,
			&a.Status, &a.DecisionReason, &created, &decided); err != nil {
			return nil, err
		}
		a.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		if decided.Valid {
			value, parseErr := parseTime(decided.String)
			if parseErr != nil {
				return nil, parseErr
			}
			a.DecidedAt = &value
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *Store) ToolCallsForProcess(ctx context.Context, processID string) ([]model.ToolCall, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT process_id,idempotency_key,action,request_hash,request_json,status,
		       COALESCE(result_json,'null'),created_at,updated_at
		FROM tool_calls WHERE process_id=? ORDER BY created_at,idempotency_key`, processID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]model.ToolCall, 0)
	for rows.Next() {
		var call model.ToolCall
		var request, output, created, updated string
		if err = rows.Scan(&call.ProcessID, &call.IdempotencyKey, &call.Action, &call.RequestHash,
			&request, &call.Status, &output, &created, &updated); err != nil {
			return nil, err
		}
		call.Request = json.RawMessage(request)
		call.Result = json.RawMessage(output)
		call.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		call.UpdatedAt, err = parseTime(updated)
		if err != nil {
			return nil, err
		}
		result = append(result, call)
	}
	return result, rows.Err()
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func parseTime(v string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, v)
}

func (s *Store) UpsertBillingCustomer(ctx context.Context, customer model.BillingCustomer) (model.BillingCustomer, error) {
	now := time.Now().UTC()
	if customer.UpdatedAt.IsZero() {
		customer.UpdatedAt = now
	}
	if customer.CreatedAt.IsZero() {
		customer.CreatedAt = customer.UpdatedAt
	}
	if strings.TrimSpace(customer.ID) == "" {
		return model.BillingCustomer{}, errors.New("billing customer id is required")
	}
	if strings.TrimSpace(customer.Email) == "" {
		return model.BillingCustomer{}, errors.New("billing customer email is required")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO customers(id,email,stripe_customer_id,created_at,updated_at)
		VALUES(?,?,?,?,?)
		ON CONFLICT(email) DO UPDATE SET
			stripe_customer_id=CASE WHEN excluded.stripe_customer_id != '' THEN excluded.stripe_customer_id ELSE customers.stripe_customer_id END,
			updated_at=excluded.updated_at`,
		customer.ID, customer.Email, customer.StripeCustomerID, formatTime(customer.CreatedAt), formatTime(customer.UpdatedAt))
	if err != nil {
		return model.BillingCustomer{}, err
	}
	return s.BillingCustomerByEmail(ctx, customer.Email)
}

func (s *Store) BillingCustomerByEmail(ctx context.Context, email string) (model.BillingCustomer, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,email,stripe_customer_id,created_at,updated_at FROM customers WHERE email=?`, email)
	return scanBillingCustomer(row)
}

func (s *Store) BillingCustomerByStripeID(ctx context.Context, stripeID string) (model.BillingCustomer, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,email,stripe_customer_id,created_at,updated_at FROM customers WHERE stripe_customer_id=?`, stripeID)
	return scanBillingCustomer(row)
}

func scanBillingCustomer(row scanner) (model.BillingCustomer, error) {
	var customer model.BillingCustomer
	var created, updated string
	err := row.Scan(&customer.ID, &customer.Email, &customer.StripeCustomerID, &created, &updated)
	if err != nil {
		return customer, err
	}
	customer.CreatedAt, err = parseTime(created)
	if err != nil {
		return customer, err
	}
	customer.UpdatedAt, err = parseTime(updated)
	return customer, err
}

func (s *Store) UpsertBillingSubscription(ctx context.Context, subscription model.BillingSubscription) error {
	if subscription.StripeSubscriptionID == "" {
		return errors.New("stripe subscription id is required")
	}
	if subscription.CustomerID == "" {
		return errors.New("billing subscription customer id is required")
	}
	if subscription.Plan == "" {
		subscription.Plan = "pro"
	}
	if subscription.Status == "" {
		subscription.Status = "unknown"
	}
	if subscription.UpdatedAt.IsZero() {
		subscription.UpdatedAt = time.Now().UTC()
	}
	var period any
	if subscription.CurrentPeriodEnd != nil {
		period = formatTime(*subscription.CurrentPeriodEnd)
	}
	cancel := 0
	if subscription.CancelAtPeriodEnd {
		cancel = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions(stripe_subscription_id,customer_id,plan,status,current_period_end,cancel_at_period_end,updated_at)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(stripe_subscription_id) DO UPDATE SET
			customer_id=excluded.customer_id,
			plan=excluded.plan,
			status=excluded.status,
			current_period_end=excluded.current_period_end,
			cancel_at_period_end=excluded.cancel_at_period_end,
			updated_at=excluded.updated_at`,
		subscription.StripeSubscriptionID, subscription.CustomerID, subscription.Plan, subscription.Status, period, cancel, formatTime(subscription.UpdatedAt))
	return err
}

func (s *Store) BillingStatus(ctx context.Context) (model.BillingStatus, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT c.id,c.email,c.stripe_customer_id,s.stripe_subscription_id,s.plan,s.status,
		       COALESCE(s.current_period_end,''),s.cancel_at_period_end,s.updated_at
		FROM subscriptions s JOIN customers c ON c.id=s.customer_id
		ORDER BY s.updated_at DESC LIMIT 1`)
	var status model.BillingStatus
	var period, updated string
	var cancel int
	err := row.Scan(&status.CustomerID, &status.Email, &status.StripeCustomerID, &status.StripeSubscriptionID,
		&status.Plan, &status.Status, &period, &cancel, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		status.Plan = "free"
		status.Status = "not_subscribed"
		status.UpdatedAt = time.Now().UTC()
		return status, nil
	}
	if err != nil {
		return status, err
	}
	if period != "" {
		parsed, parseErr := parseTime(period)
		if parseErr != nil {
			return status, parseErr
		}
		status.CurrentPeriodEnd = &parsed
	}
	status.CancelAtPeriodEnd = cancel == 1
	status.UpdatedAt, err = parseTime(updated)
	return status, err
}

func (s *Store) CreateBillingEvent(ctx context.Context, event model.BillingEvent) (bool, error) {
	if event.ProcessedAt.IsZero() {
		event.ProcessedAt = time.Now().UTC()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO billing_events(id,type,payload_hash,status,processed_at)
		VALUES(?,?,?,?,?)`, event.ID, event.Type, event.PayloadHash, event.Status, formatTime(event.ProcessedAt))
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	return changed == 1, err
}

func (s *Store) MarkBillingEvent(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE billing_events SET status=?,processed_at=? WHERE id=?`, status, formatTime(time.Now().UTC()), id)
	return err
}

func (s *Store) BillingEvent(ctx context.Context, id string) (model.BillingEvent, error) {
	var event model.BillingEvent
	var processed string
	err := s.db.QueryRowContext(ctx, `SELECT id,type,payload_hash,status,processed_at FROM billing_events WHERE id=?`, id).
		Scan(&event.ID, &event.Type, &event.PayloadHash, &event.Status, &processed)
	if err != nil {
		return event, err
	}
	event.ProcessedAt, err = parseTime(processed)
	return event, err
}
