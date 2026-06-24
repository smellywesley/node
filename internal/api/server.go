package api

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/agentos/agentos/internal/core"
	"github.com/agentos/agentos/internal/model"
)

//go:embed web/index.html
var dashboardHTML []byte

//go:embed web/app.js
var dashboardJS []byte

//go:embed web/styles.css
var dashboardCSS []byte

type Server struct {
	service      *core.Service
	operatorHash string
	approverHash string
}

func New(service *core.Service, operatorToken, approverToken string) http.Handler {
	s := &Server{
		service:      service,
		operatorHash: core.HashToken(operatorToken),
		approverHash: core.HashToken(approverToken),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.dashboard)
	mux.HandleFunc("GET /app.js", s.dashboardAsset("text/javascript; charset=utf-8", dashboardJS))
	mux.HandleFunc("GET /styles.css", s.dashboardAsset("text/css; charset=utf-8", dashboardCSS))
	mux.HandleFunc("GET /v1/health", s.health)
	mux.HandleFunc("POST /v1/processes", s.createProcess)
	mux.HandleFunc("GET /v1/processes", s.listProcesses)
	mux.HandleFunc("GET /v1/processes/{id}", s.getProcess)
	mux.HandleFunc("GET /v1/processes/{id}/events", s.events)
	mux.HandleFunc("POST /v1/processes/{id}/suspend", s.transition(model.StateSuspended))
	mux.HandleFunc("POST /v1/processes/{id}/resume", s.transition(model.StateQueued))
	mux.HandleFunc("POST /v1/processes/{id}/cancel", s.transition(model.StateCancelled))
	mux.HandleFunc("POST /v1/processes/{id}/tools", s.requestTool)
	mux.HandleFunc("POST /v1/processes/{id}/tools/{key}/start", s.startTool)
	mux.HandleFunc("POST /v1/processes/{id}/tools/{key}/complete", s.completeTool)
	mux.HandleFunc("POST /v1/processes/{id}/usage", s.updateUsage)
	mux.HandleFunc("GET /v1/processes/{id}/replay", s.replay)
	mux.HandleFunc("GET /v1/processes/{id}/audit", s.audit)
	mux.HandleFunc("GET /v1/approvals", s.approvals)
	mux.HandleFunc("POST /v1/approvals/{id}/{decision}", s.decideApproval)
	return s.secure(mux)
}

func (s *Server) secure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if isDashboardPath(r.URL.Path) {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; "+
					"img-src 'self' data:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/health" {
			next.ServeHTTP(w, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			if !sameLoopbackOrigin(r, origin) {
				writeError(w, http.StatusForbidden, "browser origin must match the loopback daemon")
				return
			}
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		actual := core.HashToken(token)
		expected := s.operatorHash
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/approvals/") {
			expected = s.approverHash
		}
		if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid local API credential")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		next.ServeHTTP(w, r)
	})
}

func isDashboardPath(path string) bool {
	return path == "/" || path == "/app.js" || path == "/styles.css"
}

func sameLoopbackOrigin(r *http.Request, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" || parsed.Host != r.Host {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(dashboardHTML)
}

func (s *Server) dashboardAsset(contentType string, content []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createProcess(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var manifest model.Manifest
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "yaml") {
		err = yaml.Unmarshal(raw, &manifest)
	} else {
		err = json.Unmarshal(raw, &manifest)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid manifest: "+err.Error())
		return
	}
	process, err := s.service.Create(r.Context(), manifest)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, process)
}

func (s *Server) listProcesses(w http.ResponseWriter, r *http.Request) {
	processes, err := s.service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range processes {
		processes[i] = sanitizeProcess(processes[i], false)
	}
	writeJSON(w, http.StatusOK, processes)
}

func (s *Server) getProcess(w http.ResponseWriter, r *http.Request) {
	process, err := s.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "process not found")
		return
	}
	writeJSON(w, http.StatusOK, sanitizeProcess(process, false))
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	events, err := s.service.Events(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) transition(state model.ProcessState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		process, err := s.service.Transition(r.Context(), r.PathValue("id"), state)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, process)
	}
}

func (s *Server) requestTool(w http.ResponseWriter, r *http.Request) {
	var request model.ToolRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	decision, err := s.service.RequestTool(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func (s *Server) completeTool(w http.ResponseWriter, r *http.Request) {
	var result model.ToolResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result.IdempotencyKey = r.PathValue("key")
	if err := s.service.CompleteTool(r.Context(), r.PathValue("id"), r.PathValue("key"), result); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

func (s *Server) startTool(w http.ResponseWriter, r *http.Request) {
	if err := s.service.StartTool(r.Context(), r.PathValue("id"), r.PathValue("key")); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) updateUsage(w http.ResponseWriter, r *http.Request) {
	var usage model.Usage
	if err := json.NewDecoder(r.Body).Decode(&usage); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.service.UpdateUsage(r.Context(), r.PathValue("id"), usage); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (s *Server) approvals(w http.ResponseWriter, r *http.Request) {
	approvals, err := s.service.Approvals(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range approvals {
		approvals[i].Payload = json.RawMessage(`{"redacted":true}`)
	}
	writeJSON(w, http.StatusOK, approvals)
}

func (s *Server) decideApproval(w http.ResponseWriter, r *http.Request) {
	decision := r.PathValue("decision")
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	approval, err := s.service.DecideApproval(r.Context(), r.PathValue("id"), decision, body.Reason)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, approval)
}

func (s *Server) replay(w http.ResponseWriter, r *http.Request) {
	state, err := s.service.Replay(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"mode": "projection", "state": state, "side_effects": false})
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	process, err := s.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	events, err := s.service.Events(r.Context(), process.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	approvals, toolCalls, err := s.service.AuditRecords(r.Context(), process.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	process = sanitizeProcess(process, true)
	for i := range events {
		switch events[i].Type {
		case "worker.stdout", "worker.stderr", "process.checkpoint", "tool.completed", "tool.failed", "tool.outcome_unknown":
			events[i].Data = json.RawMessage(`{"redacted":true}`)
		}
	}
	for i := range approvals {
		approvals[i].Payload = json.RawMessage(`{"redacted":true}`)
	}
	for i := range toolCalls {
		toolCalls[i].Request = json.RawMessage(`{"redacted":true}`)
		toolCalls[i].Result = json.RawMessage(`{"redacted":true}`)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schema_version": "v1",
		"process":        process,
		"events":         events,
		"approvals":      approvals,
		"tool_calls":     toolCalls,
		"redacted":       true,
	})
}

func redactMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key := range input {
		result[key] = "[REDACTED]"
	}
	return result
}

func sanitizeProcess(process model.Process, redactTask bool) model.Process {
	process.Manifest.Implementation.Env = redactMap(process.Manifest.Implementation.Env)
	if redactTask {
		process.Manifest.Task = "[REDACTED]"
		for i := range process.Manifest.Mounts {
			process.Manifest.Mounts[i].Source = "[REDACTED]"
		}
	}
	return process
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message, "status": status})
}

func EnsureToken(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		token := strings.TrimSpace(string(raw))
		if token == "" {
			return "", fmt.Errorf("token file %s is empty", path)
		}
		if err = secureTokenPath(path); err != nil {
			return "", err
		}
		return token, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	return writeToken(path)
}

func RotateToken(path string) (string, error) {
	return writeToken(path)
}

func writeToken(path string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	token := hex.EncodeToString(random)
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", err
	}
	if err := secureTokenPath(path); err != nil {
		return "", err
	}
	return token, nil
}
