# Billing And Metering Plan

Updated: 2026-06-24
Status: engineering plan

## Bottom Line

OpenAI API usage is pay-per-use. It is not free for a production service. If AgentOS uses your OpenAI API key for customer runs, your company pays OpenAI first and must bill customers afterward.

That means AgentOS needs two separate accounting layers:

```text
OpenAI invoice layer
  your OpenAI account pays model/tool usage

AgentOS customer billing layer
  customer/project/run ledger records billable usage
  customer pays you through Stripe, invoice, or contract
```

The dashboard's current `usage.tokens` and `usage.cost_usd` are good operational accounting. They are not yet a complete billing system.

## Pricing Modes

### Mode 1 - BYOK Alpha

Customer brings their own OpenAI API key.

Pros:

- Fastest to ship.
- You do not carry model spend risk.
- Good for local/self-hosted developers.

Cons:

- You cannot easily charge markup on model usage.
- Setup is less smooth.
- Customer sees and manages provider billing separately.

Recommended now for open-source/public alpha.

### Mode 2 - Managed Usage Pilot

AgentOS uses your provider account and bills customers for usage.

Pros:

- Smooth customer experience.
- You can charge platform fee plus usage markup.
- Customer gets one bill and one audit trail.

Cons:

- You carry abuse, runaway spend, fraud, and unpaid invoice risk.
- You need prepaid credits, hard spend caps, billing ledger, tenant isolation, and support.
- You become responsible for provider key security and usage reconciliation.

Recommended only after the local proof and GitHub PR workflow are stable.

### Mode 3 - Hybrid

Customer pays a subscription for AgentOS and either uses BYOK or buys managed usage credits.

Recommended paid structure:

```text
Free/open-source local runtime
Pro local: subscription, BYOK
Managed pilot: subscription + prepaid usage credits
Team/self-hosted: subscription + audit/history/features
Enterprise: custom support, retention, SSO, private deployment
```

## Required Billing Architecture

Minimum viable usage-based billing:

```text
process usage event
       |
       v
billable_usage ledger row
       |
       +--> customer/project/run aggregate
       |
       +--> prepaid credit decrement or invoice item
       |
       +--> spend cap enforcement
       |
       v
audit bundle includes usage source and pricing snapshot
```

Required fields for each billable row:

- `usage_id`
- `customer_id`
- `project_id`
- `process_id`
- `provider`
- `model`
- `input_tokens`
- `cached_input_tokens`
- `output_tokens`
- `tool_units`
- `pricing_snapshot_json`
- `provider_cost_usd`
- `customer_charge_usd`
- `markup_policy`
- `created_at`

Rules:

- Never compute invoices from mutable current pricing. Store the pricing snapshot used for the run.
- Never rely only on dashboard totals. Billing needs append-only rows.
- Enforce customer spend caps before starting a run and during usage updates.
- Prepaid managed usage is safer than postpaid usage during early pilots.
- Separate provider cost from customer charge so margins are visible.

## Engineering Sequence

1. Keep public alpha BYOK-first.
2. Verify live OpenAI Agents SDK demo with a real key.
3. Add a local `billable_usage` table fed by `budget.usage_updated` events.
4. Add `agentos usage --from --to --customer --project --json` for exports.
5. Add project/customer IDs to manifests or project profiles.
6. Add hard customer spend caps and prepaid credit checks.
7. Add Stripe only after local ledger and reconciliation are correct.
8. Add hosted/team billing only after tenant isolation and role controls exist.

## What Customers Should Pay For

Do not charge only for raw tokens. Tokens alone are a commodity pass-through.

Charge for:

- safe execution around agents;
- permission enforcement;
- approval workflow;
- spend caps;
- repository artifact workflow;
- audit exports;
- team history and controls.

Usage billing should cover provider cost plus risk margin. The product value is the control plane.