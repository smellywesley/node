package billing

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/agentos/agentos/internal/store"
)

type fakeStripe struct {
	checkoutRequest StripeCheckoutRequest
	portalRequest   StripePortalRequest
}

func (f *fakeStripe) CreateCheckoutSession(_ context.Context, request StripeCheckoutRequest) (StripeCheckoutSession, error) {
	f.checkoutRequest = request
	return StripeCheckoutSession{ID: "cs_test", URL: "https://checkout.stripe.com/c/pay", CustomerID: request.CustomerID}, nil
}

func (f *fakeStripe) CreatePortalSession(_ context.Context, request StripePortalRequest) (StripePortalSession, error) {
	f.portalRequest = request
	return StripePortalSession{ID: "bps_test", URL: "https://billing.stripe.com/session"}, nil
}

func TestCheckoutSessionUsesSelectedProPrice(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/billing.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stripe := &fakeStripe{}
	service := NewWithClient(db, Config{
		SecretKey: "stripe-secret", PriceProMonthly: "price_month", PriceProYearly: "price_year", PublicURL: "http://127.0.0.1:7479",
	}, stripe)
	_, err = service.CreateCheckout(context.Background(), CheckoutRequest{Email: "buyer@example.com", Interval: "yearly"})
	if err != nil {
		t.Fatal(err)
	}
	if stripe.checkoutRequest.PriceID != "price_year" || stripe.checkoutRequest.Plan != PlanPro || stripe.checkoutRequest.Interval != "yearly" {
		t.Fatalf("checkout request=%+v", stripe.checkoutRequest)
	}
	if stripe.checkoutRequest.Email != "buyer@example.com" {
		t.Fatalf("email=%q", stripe.checkoutRequest.Email)
	}
}

func TestWebhookRejectsInvalidSignatureAndDeduplicates(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/billing.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewWithClient(db, Config{WebhookSecret: "webhook-secret"}, &fakeStripe{})
	now := time.Unix(1_800_000_000, 0).UTC()
	service.now = func() time.Time { return now }
	payload := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"customer":"cus_1","customer_email":"buyer@example.com","subscription":"sub_1"}}}`)
	if _, err = service.HandleWebhook(context.Background(), payload, "t=1,v1=bad"); err == nil {
		t.Fatal("invalid signature was accepted")
	}
	header := TestSignatureHeader(payload, "webhook-secret", now)
	created, err := service.HandleWebhook(context.Background(), payload, header)
	if err != nil || !created {
		t.Fatalf("first webhook created=%v err=%v", created, err)
	}
	created, err = service.HandleWebhook(context.Background(), payload, header)
	if err != nil || created {
		t.Fatalf("duplicate webhook created=%v err=%v", created, err)
	}
	status, err := db.BillingStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Plan != PlanPro || status.Status != "active" || status.StripeSubscriptionID != "sub_1" {
		t.Fatalf("status=%+v", status)
	}
}

func TestSubscriptionWebhookUpdatesStatus(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/billing.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewWithClient(db, Config{WebhookSecret: "webhook-secret"}, &fakeStripe{})
	now := time.Unix(1_800_000_000, 0).UTC()
	service.now = func() time.Time { return now }
	payload := []byte(`{"id":"evt_2","type":"customer.subscription.updated","data":{"object":{"id":"sub_2","customer":"cus_2","status":"past_due","current_period_end":1800003600,"cancel_at_period_end":true}}}`)
	header := TestSignatureHeader(payload, "webhook-secret", now)
	if _, err = service.HandleWebhook(context.Background(), payload, header); err != nil {
		t.Fatal(err)
	}
	status, err := db.BillingStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "past_due" || !status.CancelAtPeriodEnd || status.CurrentPeriodEnd == nil {
		t.Fatalf("status=%+v", status)
	}
}

func TestRedactSecrets(t *testing.T) {
	secret := "sk_" + "test_" + strings.Repeat("a", 24)
	webhook := "whsec_" + strings.Repeat("b", 24)
	input := "STRIPE_SECRET_KEY=" + secret + " " + webhook + " https://checkout.stripe.com/c/pay/cs_test"
	redacted := RedactSecrets(input)
	if strings.Contains(redacted, secret) || strings.Contains(redacted, webhook) || strings.Contains(redacted, "checkout.stripe.com") {
		t.Fatalf("not redacted: %s", redacted)
	}
}
