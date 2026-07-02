# NODE Public Site Deployment

This directory is safe to deploy publicly. It is a Node-built static marketing and pricing site only.

It intentionally does not include:

- operator or approver tokens;
- Stripe secret keys or webhook secrets;
- SQLite state;
- worker execution endpoints;
- agent audit bundles;
- source repository contents beyond this static site.

## Cloudflare Pages

1. Create a Cloudflare Pages project.
2. Set build command to `cd deploy/public-site && npm ci && npm run build`.
3. Set output directory to `deploy/public-site/dist`.
4. Deploy.

## Netlify

1. Create a Netlify site from the repo.
2. Set build command to `cd deploy/public-site && npm ci && npm run build`.
3. Set publish directory to `deploy/public-site/dist`.

## Render

1. Create a Render Static Site from the repo, or use the root `render.yaml` Blueprint.
2. Set build command to `cd deploy/public-site && npm ci && npm run build`.
3. Set publish directory to `deploy/public-site/dist`.
4. Keep this as a static site only. Do not create a public web service for `agentos serve`.

## Vercel

1. Create a Vercel project from the repo.
2. Set build command to `cd deploy/public-site && npm ci && npm run build`.
3. Set output directory to `deploy/public-site/dist`.

## Before going live

- Add the real sales or founder email to `public/payment-links.js` as `contactEmail`.
- Connect the production domain in the hosting provider.
- If Stripe Payment Links exist, add the hosted `https://buy.stripe.com/...` URL to `public/payment-links.js`.
- If Stripe Checkout is linked through an app backend, keep the checkout endpoint server-side and keep STRIPE_SECRET_KEY server-side.
- Keep the local dashboard URL and operator token out of public docs, issue trackers, and screenshots.

## Payment Links

The static site reads provider-hosted checkout URLs from `public/payment-links.js`:

```js
window.NODE_PAYMENT_LINKS = {
  pro: "https://buy.stripe.com/...",
  enterprise: "",
  contactEmail: "sales@example.com",
  allowedHosts: ["buy.stripe.com"]
};
```

Leave a checkout value empty to keep the email fallback. Leave `contactEmail` empty only if the public site should avoid outbound lead capture until a real inbox is ready. Do not add Stripe secret keys, webhook secrets, operator tokens, raw checkout session JSON, or internal billing API URLs to this file.

## Motion Layer

The hero uses a self-hosted React/Motion bundle built by Vite. The page remains readable without JavaScript, and the Content Security Policy allows only same-origin scripts.

The live hero and the renderable video story share `src/hyperframes.js`. This keeps HUD copy, branch-light timing, signal-runner beats, and Remotion frame beats in one place.

Remotion commands:

```bash
npm run motion:compositions
npm run motion:render
```

`motion:render` outputs `out/node-signal-plane.mp4`, a branded proof/replay visual that can be used for demos, launch videos, or social cuts without adding video weight to the live page bundle.

## Important

Do not expose `agentos serve` directly on the public internet. The daemon is a local control plane and intentionally rejects non-loopback binds. A hosted app version needs tenant auth, RBAC, rate limits, audit retention policy, support bundles, and a separate runner isolation model.
