# ADR 0002: Next.js as a BFF, with opaque database-backed sessions

- Status: accepted
- Date: 2026-09-21

## Context

The web app needs authenticated calls to the Go API. A mobile app is likely
later. The options are where the credential lives (browser JS, an httpOnly
cookie on the API's domain, or an httpOnly cookie on the web app's domain) and
what the credential is (a JWT or an opaque token).

## Decision

1. **The browser only talks to Next.js.** Server Components and Server Actions
   call the Go API server-to-server with `Authorization: Bearer <token>`. The
   token lives in an httpOnly, `SameSite=Lax` cookie (`amorae_session`) on the web
   app's own domain.
2. **The token is opaque**: 32 random bytes. The API stores only its SHA-256 in
   `sessions`, with an expiry.

## Consequences

- No token is ever readable by client-side JavaScript, so XSS can't exfiltrate a
  session.
- No CORS configuration and no cross-site cookie rules, because the API is never
  called from a browser origin.
- The API stays client-agnostic. A mobile app sends the same bearer header, and
  nothing in Go changes.
- Logout and "sign out everywhere" are real: delete the rows. A JWT can't be
  revoked before it expires without adding a denylist, which is a session table
  under another name.
- A leaked database dump contains hashes, not usable tokens.
- Cost: one primary-key lookup per authenticated request. That's cheap, and the
  `SessionRepository` interface lets it move to Redis if it ever isn't.
- Cost: every API call from the web makes a server-side hop through Next.js.
  That's the price of keeping credentials out of the browser, and it's also where
  caching and response shaping for the UI can live.
- CSRF: Server Actions are POST-only, and Next.js checks their `Origin` against
  the host. `SameSite=Lax` covers cross-site form posts to any future route
  handlers as well.

## Alternatives considered

- **JWT in `localStorage`.** Rejected: readable by any injected script.
- **JWT in a cookie set by the API domain.** Rejected: brings CORS and
  cross-site cookie configuration into the picture, and still can't be revoked.
- **A third-party auth provider (Clerk, Auth0, Supabase Auth).** Reasonable, and
  maybe the right call later. It's left out of the template so the auth flow
  stays readable end to end, and so the choice of vendor (cost, data residency)
  stays open.
