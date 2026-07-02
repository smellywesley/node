package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDashboardURLKeepsCredentialOutOfQuery(t *testing.T) {
	target := dashboardURL("127.0.0.1:7467", "0123456789abcdef", "fedcba9876543210")
	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "http" || parsed.Host != "127.0.0.1:7467" || parsed.Path != "/" {
		t.Fatalf("unexpected dashboard URL: %s", target)
	}
	if parsed.RawQuery != "" {
		t.Fatalf("credential leaked into query: %s", target)
	}
	values, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	if values.Get("token") != "0123456789abcdef" {
		t.Fatalf("unexpected fragment: %q", parsed.Fragment)
	}
	if values.Has("approver_token") {
		t.Fatalf("approver credential should not be preloaded into dashboard URL: %q", parsed.Fragment)
	}
	beforeFragment := strings.Split(target, "#")[0]
	if strings.Contains(beforeFragment, "0123456789abcdef") || strings.Contains(beforeFragment, "fedcba9876543210") {
		t.Fatalf("credential appeared before URL fragment: %s", target)
	}
}

func TestValidateStateHomeRejectsBroadRoots(t *testing.T) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if err = validateStateHome(userHome); err == nil {
		t.Fatal("user profile root was accepted as AGENTOS_HOME")
	}
	if err = validateStateHome(filepath.Join(userHome, ".agentos")); err != nil {
		t.Fatalf("dedicated state directory was rejected: %v", err)
	}
	volumeRoot := filepath.VolumeName(userHome) + string(os.PathSeparator)
	if err = validateStateHome(volumeRoot); err == nil {
		t.Fatal("filesystem root was accepted as AGENTOS_HOME")
	}
}

func TestSupportBundleUsesRedactedAuditEndpointOnly(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef"
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "token"), []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	rawEndpointCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/v1/processes/proc-1/audit":
			if got := r.Header.Get("Authorization"); got != "Bearer "+token {
				t.Fatalf("unexpected authorization header: %q", got)
			}
			_, _ = w.Write([]byte(`{"process":{"task":"[redacted]"},"events":[{"type":"tool.denied","payload":"[redacted]"}]}`))
		default:
			rawEndpointCalled = true
			_, _ = w.Write([]byte(`{"leak":"raw-process-event-or-replay"}`))
		}
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "support.json")
	if err = supportBundle(config{home: home, address: serverURL.Host}, []string{"support-bundle", "proc-1", outputPath}); err != nil {
		t.Fatal(err)
	}
	if rawEndpointCalled {
		t.Fatal("support bundle called a raw process/event/replay endpoint")
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "raw-process-event-or-replay") {
		t.Fatalf("support bundle included raw endpoint data: %s", string(raw))
	}
	var bundle map[string]any
	if err = json.Unmarshal(raw, &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle["redacted"] != true {
		t.Fatalf("support bundle was not marked redacted: %#v", bundle["redacted"])
	}
	if _, ok := bundle["audit"]; !ok {
		t.Fatalf("support bundle missing redacted audit export: %#v", bundle)
	}
	if _, ok := bundle["events"]; ok {
		t.Fatalf("support bundle should not include raw events: %#v", bundle)
	}
	if _, ok := bundle["replay"]; ok {
		t.Fatalf("support bundle should not include raw replay: %#v", bundle)
	}
}

func TestSupportBundleFailsWhenRedactedAuditFails(t *testing.T) {
	token := "0123456789abcdef0123456789abcdef"
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "token"), []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/health":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/v1/processes/proc-1/audit":
			http.Error(w, `{"error":"audit unavailable"}`, http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "support.json")
	err = supportBundle(config{home: home, address: serverURL.Host}, []string{"support-bundle", "proc-1", outputPath})
	if err == nil {
		t.Fatal("support bundle succeeded without a redacted audit export")
	}
	if !strings.Contains(err.Error(), "support bundle incomplete") || !strings.Contains(err.Error(), "audit") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("support bundle wrote incomplete output, stat err: %v", statErr)
	}
}

func TestSupportGuidanceExplainsDockerRescue(t *testing.T) {
	guidance := supportGuidance(diagnostic{Name: "docker", Status: "WARN", Detail: "not found"})
	for _, want := range []string{"problem:", "cause:", "fix:", "Docker Desktop", "doctor --support"} {
		if !strings.Contains(guidance, want) {
			t.Fatalf("guidance %q missing %q", guidance, want)
		}
	}
}
