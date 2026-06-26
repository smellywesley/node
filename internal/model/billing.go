package model

import "time"

type BillingPlan struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Audience        string   `json:"audience"`
	PriceLabel      string   `json:"price_label"`
	AnnualLabel     string   `json:"annual_label,omitempty"`
	BillingMode     string   `json:"billing_mode"`
	ModelUsage      string   `json:"model_usage"`
	CheckoutEnabled bool     `json:"checkout_enabled"`
	Features        []string `json:"features"`
}

type PricingResponse struct {
	PaymentProvider string        `json:"payment_provider"`
	CheckoutMode    string        `json:"checkout_mode"`
	Plans           []BillingPlan `json:"plans"`
	ManagedUsage    string        `json:"managed_usage"`
}

type BillingCustomer struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	StripeCustomerID string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type BillingSubscription struct {
	CustomerID           string     `json:"customer_id"`
	StripeSubscriptionID string     `json:"stripe_subscription_id"`
	Plan                 string     `json:"plan"`
	Status               string     `json:"status"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool       `json:"cancel_at_period_end"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type BillingStatus struct {
	Plan                 string     `json:"plan"`
	Status               string     `json:"status"`
	CustomerID           string     `json:"customer_id,omitempty"`
	Email                string     `json:"email,omitempty"`
	StripeCustomerID     string     `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string     `json:"stripe_subscription_id,omitempty"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool       `json:"cancel_at_period_end"`
	CheckoutConfigured   bool       `json:"checkout_configured"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type BillingEvent struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	PayloadHash string    `json:"payload_hash"`
	Status      string    `json:"status"`
	ProcessedAt time.Time `json:"processed_at"`
}
