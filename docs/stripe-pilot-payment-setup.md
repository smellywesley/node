# Stripe Pilot Payment Setup

Updated: 2026-07-06
Status: use for qualified private pilots only; not self-serve hosted SaaS billing

## Purpose

Now that a Stripe account exists, use it to collect payment after a buyer has passed the pilot-fit questions and accepted the local/private boundary. Do not use Stripe to imply NODE is already a hosted multi-tenant platform, managed model-credit product, SOC 2 certified product, or HIPAA-ready product.

The current commercial motion is:

```text
public site -> pilot-fit questions -> Calendly founder proof call -> written pilot scope -> Stripe payment link or invoice -> local/private proof session
```

## Safe Stripe Modes Today

### 1. Stripe Payment Link for a qualified pilot

Use this first.

Create a Stripe Payment Link for one pilot package, for example:

- name: NODE Private Pilot
- price: S$750 for the entry controlled-workflow proof, or the buyer-specific amount from the proposal for larger enterprise scope
- description: Founder-led local/private AI-agent control proof: one workflow, policy gate, sandbox run, spend cap, approval path, replay, and redacted audit bundle
- tax/invoice settings: configure in Stripe based on your Stripe account and jurisdiction

Then set the public-site environment variable:

```text
NODE_PUBLIC_PILOT_PAYMENT_LINK=https://buy.stripe.com/bJeeVfc0h6cJ95m9dK7g400
```

Keep the Calendly fit-check URL active too:

```text
NODE_PUBLIC_PILOT_CONTACT_URL=https://calendly.com/wesleyong2004/node-pilot-fit-check
NODE_PUBLIC_CONTACT_EMAIL=owowesley@gmail.com
```

### 2. Stripe invoice after the proof call

Use invoices when the buyer needs procurement, a custom scope, or a company billing email. This is usually better for larger private pilots.

### 3. Runtime Stripe Checkout for BYOK Pro

The local daemon has a Stripe-hosted checkout slice, but it is not the same as hosted SaaS billing. Use it only for BYOK subscription experiments after webhook configuration, subscription-state testing, and entitlement checks are verified.

Required local runtime variables for that path are intentionally private and must never be committed:

```text
STRIPE_SECRET_KEY=sk_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_PRO_MONTHLY=price_...
STRIPE_PRICE_PRO_YEARLY=price_...
APP_PUBLIC_URL=https://...
```

## Render/Public Site Variables

For the static public site, configure only public-safe values in Render or the deployment provider:

```text
NODE_PUBLIC_CONTACT_EMAIL=owowesley@gmail.com
NODE_PUBLIC_PILOT_CONTACT_URL=https://calendly.com/wesleyong2004/node-pilot-fit-check
NODE_PUBLIC_PILOT_PAYMENT_LINK=https://buy.stripe.com/bJeeVfc0h6cJ95m9dK7g400
NODE_PUBLIC_PROOF_DEMO_URL=https://...
```

Never set or expose these in the static site:

```text
STRIPE_SECRET_KEY
STRIPE_WEBHOOK_SECRET
AGENTOS_OPERATOR_TOKEN
AGENTOS_APPROVER_TOKEN
OPENAI_API_KEY
```

## Verification

After setting the public variables, run:

```powershell
cd deploy\public-site
npm run configure:cta
npm run test:cta
npm run build
cd ..\..
.\scripts\test-pilot-readiness.cmd -AllowBlockers
```

Expected improvement:

- `Generated env CTA has real pilot path` should pass when contact email, Calendly URL, or Stripe payment link is configured.
- `Public proof demo URL is configured` should pass only after the proof video URL is set.
- Docker-related checks still require Docker Desktop to be running.

## Buyer-Safe Payment Copy

Use this framing:

> Payment reserves a founder-led private/local pilot. NODE will run one controlled workflow with policy gates, sandboxing, cost cap, approval path, replay, and a redacted audit bundle. Hosted multi-tenant backend, managed model credits, SOC 2 certification, and HIPAA readiness are not included in this pilot.

## Do Not Do Yet

- Do not sell managed model credits until tenant isolation, spend caps, and a billable usage ledger exist.
- Do not use PHI in demos or pilots.
- Do not claim SOC 2 or HIPAA compliance.
- Do not publish runtime Stripe secret keys or webhook secrets to the static public site.
- Do not treat a successful Payment Link as proof of subscription entitlement inside the local runtime.
