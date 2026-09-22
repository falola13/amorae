import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const SESSION_COOKIE = "amorae_session";

// Optimistic guard only. A missing cookie is a reliable signal; a present
// cookie proves nothing (it may be expired or revoked), so the signed-in
// shell re-checks with GET /users/me and sends a 401 back through /login.
//
// It never redirects a logged-in-looking visitor away from /login or
// /welcome, so a stale cookie can't trap anyone in a loop.
const PUBLIC = ["/welcome", "/login", "/register", "/offline", "/terms", "/privacy", "/api/", "/_next/", "/icons/", "/splash/", "/brand/", "/manifest.webmanifest", "/sw.js", "/robots.txt", "/favicon"];

export function proxy(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  if (PUBLIC.some((p) => pathname === p || pathname.startsWith(p))) return NextResponse.next();
  if (request.cookies.has(SESSION_COOKIE)) return NextResponse.next();

  // Home goes to the welcome screen; deep links go to login and come back.
  if (pathname === "/") return NextResponse.redirect(new URL("/welcome", request.url));
  const loginUrl = new URL("/login", request.url);
  loginUrl.searchParams.set("next", pathname + search);
  return NextResponse.redirect(loginUrl);
}

export const config = {
  matcher: ["/((?!_next/static|_next/image).*)"],
};
