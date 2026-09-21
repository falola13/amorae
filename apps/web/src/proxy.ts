import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const SESSION_COOKIE = "amorae_session";

// Optimistic guard only. A *missing* cookie is a reliable signal (bounce to
// /login), but a *present* cookie proves nothing — it could be expired or
// revoked, and Proxy can't call the API to check. The dashboard page is the
// real gate: it calls getCurrentUser() and redirects to
// /login?reason=expired on a 401.
//
// This never redirects an already-logged-in-looking visitor away from
// /login — if it did, a stale cookie plus a dead API session would loop
// forever (login can't be reached to clear the cookie that's causing the
// loop). Logging in simply overwrites the cookie instead.
export function proxy(request: NextRequest) {
  if (request.cookies.has(SESSION_COOKIE)) {
    return NextResponse.next();
  }

  // Keep the query string (e.g. the PWA's ?source=pwa, or a deep link's
  // ?tab=...) so the user lands exactly where they were headed. The login
  // action re-validates `next` before redirecting to it.
  const loginUrl = new URL("/login", request.url);
  loginUrl.searchParams.set("next", request.nextUrl.pathname + request.nextUrl.search);
  return NextResponse.redirect(loginUrl);
}

export const config = {
  matcher: ["/dashboard", "/dashboard/:path*"],
};
