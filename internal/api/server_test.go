package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agentos/agentos/internal/billing"
	"github.com/agentos/agentos/internal/core"
	"github.com/agentos/agentos/internal/model"
	"github.com/agentos/agentos/internal/runner"
	"github.com/agentos/agentos/internal/store"
)

type immediateRunner struct{}

func (immediateRunner) Run(context.Context, model.Process, runner.ToolHandler, runner.UsageHandler, func(string, any)) (runner.Output, error) {
	return runner.Output{}, nil
}
func (immediateRunner) Cancel(context.Context, string) error { return nil }

func TestAuthenticationAndOriginProtection(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := core.New(db, immediateRunner{}, 1)
	defer service.Close()
	handler := New(service, "secret-token", "approver-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/processes", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", response.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/processes", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Origin", "https://attacker.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusForbidden {
		t.Fatalf("origin status=%d", response.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/processes", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("authenticated status=%d body=%s", response.Code, response.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/processes", nil)
	req.Host = "127.0.0.1:7467"
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Origin", "http://127.0.0.1:7467")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("same-origin dashboard request status=%d body=%s", response.Code, response.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/approvals/missing/approved", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("operator token approved action, status=%d", response.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/approvals/missing/approved", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer approver-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code == http.StatusUnauthorized {
		t.Fatalf("approver token was rejected: %s", response.Body.String())
	}
}

func TestDashboardAssetsArePublicAndHardened(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := core.New(db, immediateRunner{}, 1)
	defer service.Close()
	handler := New(service, "secret-token", "approver-token")

	for _, test := range []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/", contentType: "text/html", contains: "NODE Enterprise Control Plane"},
		{path: "/app.js", contentType: "text/javascript", contains: "agentos.operatorToken"},
		{path: "/styles.css", contentType: "text/css", contains: "--accent"},
	} {
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", test.path, response.Code, response.Body.String())
		}
		if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, test.contentType) {
			t.Fatalf("%s content-type=%q", test.path, contentType)
		}
		if !strings.Contains(response.Body.String(), test.contains) {
			t.Fatalf("%s body missing %q", test.path, test.contains)
		}
		if response.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("%s missing content security policy", test.path)
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s cache-control=%q", test.path, response.Header().Get("Cache-Control"))
		}
	}
}

func TestAuditExportIncludesRedactedControlRecords(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	process := model.Process{
		ID: "process-1", Name: "audit", State: model.StateCreated, CreatedAt: now, UpdatedAt: now,
		Manifest: model.Manifest{
			Image: "example", Task: "sensitive task",
			Implementation: model.Implementation{Command: []string{"run"}, Env: map[string]string{"TOKEN": "secret"}},
		},
	}
	if err = db.CreateProcess(context.Background(), process); err != nil {
		t.Fatal(err)
	}
	if _, err = db.CreateToolCall(context.Background(), process.ID, "key-1", "fs.write", "hash-1",
		map[string]string{"secret": "value"}); err != nil {
		t.Fatal(err)
	}
	approval := model.Approval{
		ID: "approval-1", ProcessID: process.ID, IdempotencyKey: "key-1", Action: "fs.write",
		Payload: json.RawMessage(`{"secret":"value"}`), Status: "pending", CreatedAt: now,
	}
	if err = db.CreateApproval(context.Background(), approval, "hash-1", "v1"); err != nil {
		t.Fatal(err)
	}
	if err = db.AppendEvent(context.Background(), process.ID, "process.execution_error", map[string]any{"error": "raw runtime secret"}); err != nil {
		t.Fatal(err)
	}
	service := core.New(db, immediateRunner{}, 1)
	defer service.Close()
	handler := New(service, "secret-token", "approver-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/processes/"+process.ID+"/audit", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{`"approvals"`, `"tool_calls"`, `"request":{"redacted":true}`, `"payload":{"redacted":true}`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("audit body missing %s: %s", expected, body)
		}
	}
	for _, secret := range []string{"sensitive task", `"secret":"value"`, "raw runtime secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("audit body leaked %q: %s", secret, body)
		}
	}
}

type apiFakeStripe struct {
	checkout billing.StripeCheckoutRequest
}

func (f *apiFakeStripe) CreateCheckoutSession(_ context.Context, request billing.StripeCheckoutRequest) (billing.StripeCheckoutSession, error) {
	f.checkout = request
	return billing.StripeCheckoutSession{ID: "cs_test", URL: "https://checkout.stripe.com/c/session"}, nil
}

func (f *apiFakeStripe) CreatePortalSession(_ context.Context, request billing.StripePortalRequest) (billing.StripePortalSession, error) {
	return billing.StripePortalSession{ID: "bps_test", URL: "https://billing.stripe.com/session"}, nil
}

func TestBillingAPIRequiresAuthAndCreatesCheckout(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := core.New(db, immediateRunner{}, 1)
	defer service.Close()
	stripe := &apiFakeStripe{}
	billingService := billing.NewWithClient(db, billing.Config{
		SecretKey: "stripe-secret", PriceProMonthly: "price_month", PriceProYearly: "price_year", PublicURL: "http://127.0.0.1:7479",
	}, stripe)
	handler := NewWithBilling(service, "secret-token", "approver-token", billingService)

	req := httptest.NewRequest(http.MethodGet, "/v1/billing/status", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated billing status=%d", response.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/billing/portal", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer secret-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusConflict {
		t.Fatalf("portal without customer status=%d body=%s", response.Code, response.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/billing/checkout", strings.NewReader(`{"email":"buyer@example.com","interval":"monthly"}`))
	req.Header.Set("Authorization", "Bearer secret-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("checkout status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, "checkout.stripe.com") || strings.Contains(body, "stripe-secret") {
		t.Fatalf("unexpected checkout body: %s", body)
	}
	if stripe.checkout.PriceID != "price_month" {
		t.Fatalf("checkout request=%+v", stripe.checkout)
	}
}

func TestBillingWebhookUsesStripeSignatureNotOperatorToken(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := core.New(db, immediateRunner{}, 1)
	defer service.Close()
	billingService := billing.NewWithClient(db, billing.Config{WebhookSecret: "webhook-secret"}, &apiFakeStripe{})
	handler := NewWithBilling(service, "secret-token", "approver-token", billingService)
	payload := []byte(`{"id":"evt_api","type":"checkout.session.completed","data":{"object":{"customer":"cus_api","customer_email":"buyer@example.com","subscription":"sub_api"}}}`)
	now := time.Now().UTC()
	req := httptest.NewRequest(http.MethodPost, "/v1/billing/webhook", strings.NewReader(string(payload)))
	req.Header.Set("Stripe-Signature", billing.TestSignatureHeader(payload, "webhook-secret", now))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("webhook status=%d body=%s", response.Code, response.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/billing/status", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "sub_api") {
		t.Fatalf("billing status=%d body=%s", response.Code, response.Body.String())
	}
}
