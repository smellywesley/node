package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/agentos/agentos/internal/model"
	"github.com/agentos/agentos/internal/store"
)

const (
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

type Config struct {
	SecretKey       string
	WebhookSecret   string
	PriceProMonthly string
	PriceProYearly  string
	PublicURL       string
}

func ConfigFromEnv() Config {
	publicURL := strings.TrimRight(os.Getenv("APP_PUBLIC_URL"), "/")
	if publicURL == "" {
		publicURL = "http://127.0.0.1:7479"
	}
	return Config{
		SecretKey:       strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
		WebhookSecret:   strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")),
		PriceProMonthly: strings.TrimSpace(os.Getenv("STRIPE_PRICE_PRO_MONTHLY")),
		PriceProYearly:  strings.TrimSpace(os.Getenv("STRIPE_PRICE_PRO_YEARLY")),
		PublicURL:       publicURL,
	}
}

func (c Config) CheckoutConfigured() bool {
	return c.SecretKey != "" && c.PublicURL != "" && c.PriceProMonthly != "" && c.PriceProYearly != ""
}

type StripeClient interface {
	CreateCheckoutSession(context.Context, StripeCheckoutRequest) (StripeCheckoutSession, error)
	CreatePortalSession(context.Context, StripePortalRequest) (StripePortalSession, error)
}

type StripeCheckoutRequest struct {
	PriceID    string
	CustomerID string
	Email      string
	SuccessURL string
	CancelURL  string
	Plan       string
	Interval   string
}

type StripeCheckoutSession struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	CustomerID string `json:"customer"`
}

type StripePortalRequest struct {
	CustomerID string
	ReturnURL  string
}

type StripePortalSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type HTTPStripeClient struct {
	SecretKey string
	BaseURL   string
	Client    *http.Client
}

func (c HTTPStripeClient) CreateCheckoutSession(ctx context.Context, request StripeCheckoutRequest) (StripeCheckoutSession, error) {
	values := url.Values{}
	values.Set("mode", "subscription")
	values.Set("success_url", request.SuccessURL)
	values.Set("cancel_url", request.CancelURL)
	values.Set("line_items[0][price]", request.PriceID)
	values.Set("line_items[0][quantity]", "1")
	values.Set("metadata[plan]", request.Plan)
	values.Set("metadata[interval]", request.Interval)
	if request.CustomerID != "" {
		values.Set("customer", request.CustomerID)
	} else {
		values.Set("customer_email", request.Email)
	}
	var session StripeCheckoutSession
	if err := c.postForm(ctx, "/v1/checkout/sessions", values, &session); err != nil {
		return session, err
	}
	if session.URL == "" {
		return session, errors.New("Stripe checkout session did not include a hosted URL")
	}
	return session, nil
}

func (c HTTPStripeClient) CreatePortalSession(ctx context.Context, request StripePortalRequest) (StripePortalSession, error) {
	values := url.Values{}
	values.Set("customer", request.CustomerID)
	values.Set("return_url", request.ReturnURL)
	var session StripePortalSession
	if err := c.postForm(ctx, "/v1/billing_portal/sessions", values, &session); err != nil {
		return session, err
	}
	if session.URL == "" {
		return session, errors.New("Stripe portal session did not include a hosted URL")
	}
	return session, nil
}

func (c HTTPStripeClient) postForm(ctx context.Context, path string, values url.Values, target any) error {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://api.stripe.com"
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+path, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Stripe API returned %s: %s", resp.Status, RedactSecrets(string(body)))
	}
	return json.Unmarshal(body, target)
}

type Service struct {
	store  *store.Store
	config Config
	stripe StripeClient
	now    func() time.Time
}

func New(store *store.Store, config Config) *Service {
	return NewWithClient(store, config, HTTPStripeClient{SecretKey: config.SecretKey})
}

func NewWithClient(store *store.Store, config Config, client StripeClient) *Service {
	return &Service{store: store, config: config, stripe: client, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Pricing() model.PricingResponse {
	checkoutReady := s.config.CheckoutConfigured()
	return model.PricingResponse{
		PaymentProvider: "stripe",
		CheckoutMode:    "hosted_checkout",
		ManagedUsage:    "locked until tenant isolation, spend caps, and billable_usage ledger are complete",
		Plans: []model.BillingPlan{
			{
				ID: "free", Name: "Free Local", Audience: "Local developers", PriceLabel: "$0", BillingMode: "BYOK",
				ModelUsage: "Customer provider key", CheckoutEnabled: false,
				Features: []string{"Local demo", "Replayable process history", "Manual audit export", "No team history"},
			},
			{
				ID: "pro", Name: "Pro", Audience: "Agent builders and small teams", PriceLabel: "$99/operator/month", AnnualLabel: "$948/operator/year", BillingMode: "Subscription + BYOK",
				ModelUsage: "Customer provider key; no model credits included", CheckoutEnabled: checkoutReady,
				Features: []string{"Hosted Stripe checkout", "Audit bundles", "Replay", "Policy templates", "GitHub artifact workflow when ready"},
			},
			{
				ID: "enterprise", Name: "Enterprise", Audience: "Security-conscious teams", PriceLabel: "Starts at $2,500/month", AnnualLabel: "Starts at $30k/year", BillingMode: "Annual contract",
				ModelUsage: "BYOK or negotiated managed usage later", CheckoutEnabled: false,
				Features: []string{"Private deployment", "SSO/RBAC roadmap", "Longer audit retention", "Support SLA", "Security review"},
			},
		},
	}
}

func (s *Service) Status(ctx context.Context) (model.BillingStatus, error) {
	status, err := s.store.BillingStatus(ctx)
	status.CheckoutConfigured = s.config.CheckoutConfigured()
	return status, err
}

type CheckoutRequest struct {
	Email    string `json:"email"`
	Interval string `json:"interval"`
}

type HostedSession struct {
	URL string `json:"url"`
}

func (s *Service) CreateCheckout(ctx context.Context, request CheckoutRequest) (HostedSession, error) {
	if !s.config.CheckoutConfigured() {
		return HostedSession{}, errors.New("Stripe checkout is not configured on this server")
	}
	email := strings.ToLower(strings.TrimSpace(request.Email))
	if !validEmail(email) {
		return HostedSession{}, errors.New("a valid billing email is required")
	}
	interval := strings.ToLower(strings.TrimSpace(request.Interval))
	priceID := s.config.PriceProMonthly
	if interval == "yearly" || interval == "annual" {
		interval = "yearly"
		priceID = s.config.PriceProYearly
	} else {
		interval = "monthly"
	}
	customer, err := s.store.UpsertBillingCustomer(ctx, model.BillingCustomer{ID: uuid.NewString(), Email: email, UpdatedAt: s.now()})
	if err != nil {
		return HostedSession{}, err
	}
	session, err := s.stripe.CreateCheckoutSession(ctx, StripeCheckoutRequest{
		PriceID: priceID, CustomerID: customer.StripeCustomerID, Email: email,
		SuccessURL: s.config.PublicURL + "/#billing-title", CancelURL: s.config.PublicURL + "/#billing-title",
		Plan: PlanPro, Interval: interval,
	})
	if err != nil {
		return HostedSession{}, err
	}
	return HostedSession{URL: session.URL}, nil
}

func (s *Service) CreatePortal(ctx context.Context) (HostedSession, error) {
	status, err := s.store.BillingStatus(ctx)
	if err != nil {
		return HostedSession{}, err
	}
	if status.StripeCustomerID == "" {
		return HostedSession{}, errors.New("no Stripe customer is linked yet; complete checkout first")
	}
	session, err := s.stripe.CreatePortalSession(ctx, StripePortalRequest{CustomerID: status.StripeCustomerID, ReturnURL: s.config.PublicURL + "/#billing-title"})
	if err != nil {
		return HostedSession{}, err
	}
	return HostedSession{URL: session.URL}, nil
}

func (s *Service) HandleWebhook(ctx context.Context, payload []byte, signature string) (bool, error) {
	if s.config.WebhookSecret == "" {
		return false, errors.New("Stripe webhook secret is not configured")
	}
	if err := verifySignature(payload, signature, s.config.WebhookSecret, s.now()); err != nil {
		return false, err
	}
	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return false, err
	}
	if event.ID == "" || event.Type == "" {
		return false, errors.New("invalid Stripe event")
	}
	payloadHash := hashPayload(payload)
	created, err := s.store.CreateBillingEvent(ctx, model.BillingEvent{ID: event.ID, Type: event.Type, PayloadHash: payloadHash, Status: "received", ProcessedAt: s.now()})
	if err != nil || !created {
		return created, err
	}
	status := "ignored"
	switch event.Type {
	case "checkout.session.completed":
		status, err = s.handleCheckoutCompleted(ctx, event.Data.Object)
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		status, err = s.handleSubscription(ctx, event.Data.Object)
	}
	if err != nil {
		_ = s.store.MarkBillingEvent(ctx, event.ID, "failed")
		return true, err
	}
	return true, s.store.MarkBillingEvent(ctx, event.ID, status)
}

func (s *Service) handleCheckoutCompleted(ctx context.Context, object map[string]any) (string, error) {
	customerID := stringValue(object["customer"])
	subscriptionID := stringValue(object["subscription"])
	email := strings.ToLower(firstNonEmpty(
		stringValue(object["customer_email"]),
		stringValue(nested(object, "customer_details", "email")),
	))
	if customerID == "" || subscriptionID == "" {
		return "ignored", nil
	}
	if email == "" {
		email = "unknown+stripe@node.local"
	}
	customer, err := s.store.UpsertBillingCustomer(ctx, model.BillingCustomer{ID: uuid.NewString(), Email: email, StripeCustomerID: customerID, UpdatedAt: s.now()})
	if err != nil {
		return "failed", err
	}
	return "processed", s.store.UpsertBillingSubscription(ctx, model.BillingSubscription{
		CustomerID: customer.ID, StripeSubscriptionID: subscriptionID, Plan: PlanPro, Status: "active", UpdatedAt: s.now(),
	})
}

func (s *Service) handleSubscription(ctx context.Context, object map[string]any) (string, error) {
	customerID := stringValue(object["customer"])
	subscriptionID := stringValue(object["id"])
	if customerID == "" || subscriptionID == "" {
		return "ignored", nil
	}
	customer, err := s.store.BillingCustomerByStripeID(ctx, customerID)
	if err != nil {
		customer, err = s.store.UpsertBillingCustomer(ctx, model.BillingCustomer{ID: uuid.NewString(), Email: "unknown+stripe@node.local", StripeCustomerID: customerID, UpdatedAt: s.now()})
		if err != nil {
			return "failed", err
		}
	}
	var periodEnd *time.Time
	if raw := int64Value(object["current_period_end"]); raw > 0 {
		value := time.Unix(raw, 0).UTC()
		periodEnd = &value
	}
	status := stringValue(object["status"])
	if status == "" {
		status = "unknown"
	}
	return "processed", s.store.UpsertBillingSubscription(ctx, model.BillingSubscription{
		CustomerID: customer.ID, StripeSubscriptionID: subscriptionID, Plan: PlanPro, Status: status,
		CurrentPeriodEnd: periodEnd, CancelAtPeriodEnd: boolValue(object["cancel_at_period_end"]), UpdatedAt: s.now(),
	})
}

type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object map[string]any `json:"object"`
	} `json:"data"`
}

func verifySignature(payload []byte, header, secret string, now time.Time) error {
	values := map[string][]string{}
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok {
			values[key] = append(values[key], value)
		}
	}
	if len(values["t"]) == 0 || len(values["v1"]) == 0 {
		return errors.New("missing Stripe signature")
	}
	timestamp, err := strconv.ParseInt(values["t"][0], 10, 64)
	if err != nil {
		return errors.New("invalid Stripe signature timestamp")
	}
	if delta := now.Sub(time.Unix(timestamp, 0)); delta > 5*time.Minute || delta < -5*time.Minute {
		return errors.New("Stripe signature timestamp is outside tolerance")
	}
	expected := signPayload(timestamp, payload, secret)
	for _, candidate := range values["v1"] {
		decoded, err := hex.DecodeString(candidate)
		if err == nil && hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return errors.New("invalid Stripe signature")
}

func TestSignatureHeader(payload []byte, secret string, timestamp time.Time) string {
	return fmt.Sprintf("t=%d,v1=%s", timestamp.Unix(), hex.EncodeToString(signPayload(timestamp.Unix(), payload, secret)))
}

func signPayload(timestamp int64, payload []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return mac.Sum(nil)
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`STRIPE_[A-Z0-9_]*=([^\s]+)`),
	regexp.MustCompile(`sk_(test|live)_[A-Za-z0-9_]+`),
	regexp.MustCompile(`whsec_[A-Za-z0-9_]+`),
	regexp.MustCompile(`https://checkout\.stripe\.com/[A-Za-z0-9_/?=&.%-]+`),
}

func RedactSecrets(input string) string {
	output := input
	for _, pattern := range secretPatterns {
		output = pattern.ReplaceAllString(output, "[REDACTED]")
	}
	return output
}

func validEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email[strings.LastIndex(email, "@"):], ".")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nested(object map[string]any, keys ...string) any {
	var current any = object
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case json.Number:
		result, _ := v.Int64()
		return result
	default:
		return 0
	}
}

func boolValue(value any) bool {
	v, _ := value.(bool)
	return v
}

func DecodeWebhookBody(reader io.Reader) ([]byte, error) {
	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, io.LimitReader(reader, 1<<20))
	return buffer.Bytes(), err
}
