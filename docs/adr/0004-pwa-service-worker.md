# ADR 0004: Hand-written service worker that never caches personal data

- Status: accepted
- Date: 2026-09-21

## Context

Amorae ships as an installable PWA: home-screen icon, splash screens, and
standalone display. Installability needs a manifest and a service worker.
The open question is what the worker should cache.

Every page is rendered on the server for the signed-in user, behind an
httpOnly session cookie (ADR 0002). Amorae is also likely to live on phones
that two people share.

## Decision

1. **Never cache HTML, RSC payloads, or API responses.** Page navigations go
   to the network every time. When the network fails, the worker serves a
   precached, static `/offline` page, at the URL the user asked for, so
   "Try again" retries that exact page.
2. **Cache only what's identical for every visitor.** That means
   content-hashed `/_next/static/*` files (cache first, trimmed to 200
   entries) and brand assets under `/icons`, `/splash` and `/brand`, plus the
   manifest (stale-while-revalidate).
3. **Leave non-GET requests alone.** Server Actions are POSTs and pass straight
   through, as do cross-origin requests.
4. **Hand-write it (`public/sw.js`, about 150 lines)** rather than adopt Serwist or
   Workbox.
5. Register it in production builds only. In development, actively unregister
   it, because `docker compose` and `npm run dev` share `localhost:3000`.

## Consequences

- No cached page can show one person's data to someone else, including after
  logout. That's verified in a browser: after loading a signed-in dashboard,
  the caches hold only static files, the offline page and the manifest.
- There's no offline *reading* of data. Opening the app with no signal shows the
  offline page, not the last-seen dashboard. If offline reading becomes a
  requirement, it needs per-user encrypted storage cleared on logout, not a
  shared HTTP cache. That calls for a new ADR.
- `sw.js` has no build-time hashing, so **bump `VERSION` in `public/sw.js` when
  the worker or the `/offline` page changes.** That's the only way installed
  apps pick up the change and clear old caches.
- One file, no build plugin, readable top to bottom. The cost is no precache
  manifest, which we don't need while pages are never cached.

## Alternatives considered

- **Serwist (the maintained next-pwa successor).** It's good, and the
  recommended path if offline caching grows. For now it adds a build plugin
  and a generated precache manifest to protect a policy whose whole point is
  to cache almost nothing.
- **Next's `experimental.useOffline`.** It retries soft navigations and Server
  Actions when the connection drops. It's worth adopting once it's stable. It
  complements the worker rather than replacing it, because it can't cover a
  cold start with no network.
