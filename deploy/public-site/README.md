# NODE Public Site Deployment

This directory is safe to deploy publicly. It is a static marketing and pricing site only.

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

## Vercel

1. Create a Vercel project from the repo.
2. Set output directory to `deploy/public-site`.
3. No framework preset is required.

## Before going live

- Replace hello@your-domain.com in index.html with the real sales or founder email.
- Connect the production domain in the hosting provider.
- If Stripe Checkout is later linked from this page, use publishable keys only in browser code and keep STRIPE_SECRET_KEY server-side.
- Keep the local dashboard URL and operator token out of public docs, issue trackers, and screenshots.

## Important

Do not expose `agentos serve` directly on the public internet. The daemon is a local control plane and intentionally rejects non-loopback binds. A hosted app version needs tenant auth, RBAC, rate limits, audit retention policy, support bundles, and a separate runner isolation model.
