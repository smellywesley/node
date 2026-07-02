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
2. Set build command to empty / none.
3. Set output directory to `deploy/public-site`.
4. Deploy.

## Netlify

1. Create a Netlify site from the repo.
2. Set publish directory to `deploy/public-site`.
3. No build command is required.

## Render

1. Create a Render Static Site from the repo, or use the root `render.yaml` Blueprint.
2. Set build command to `cd deploy/public-site && npm ci && npm run build`.
3. Set publish directory to `deploy/public-site/dist`.
4. Keep this as a static site only. Do not create a public web service for `agentos serve`.

## Vercel

1. Create a Vercel project from the repo.
2. Set output directory to `deploy/public-site`.
3. No framework preset is required.

## Before going live

- Replace hello@your-domain.com in index.html with the real sales or founder email.
- Connect the production domain in the hosting provider.
- If Stripe Payment Links exist, replace the Pro CTA with the hosted Payment Link.
- If Stripe Checkout is linked through an app backend, keep the checkout endpoint server-side and keep STRIPE_SECRET_KEY server-side.
- Keep the local dashboard URL and operator token out of public docs, issue trackers, and screenshots.

## Motion Layer

The hero uses a self-hosted React/Motion bundle built by Vite. The page remains readable without JavaScript, and the Content Security Policy allows only same-origin scripts.

## Important

Do not expose `agentos serve` directly on the public internet. The daemon is a local control plane and intentionally rejects non-loopback binds. A hosted app version needs tenant auth, RBAC, rate limits, audit retention policy, support bundles, and a separate runner isolation model.
