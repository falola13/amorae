import { NextResponse, type NextRequest } from "next/server";

import { errorBody, NETWORK_ERROR_MESSAGE } from "@/lib/api/envelope";
import { visitorHeaders } from "@/lib/api/upstream";
import { getSessionToken } from "@/lib/auth/session";
import { env } from "@/lib/env";

// Same-origin front door for browser calls. The axios client (lib/api/http.ts)
// hits /api/v1/<path>; this forwards to <API_URL>/v1/<path> with the session
// cookie's bearer token attached. The browser never holds the token
// (docs/adr/0002). Nothing here is cached: every call carries a session.

const TIMEOUT_MS = 10_000;
const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

async function forward(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  if (!SAFE_METHODS.has(request.method) && isCrossOrigin(request)) {
    return NextResponse.json(errorBody("forbidden", "This request came from another site."), { status: 403 });
  }

  const path = upstreamPath((await params).path);
  if (path === null) {
    return NextResponse.json(errorBody("route_not_found", "That page doesn't exist."), { status: 404 });
  }

  const target = new URL(`${env.API_URL}/v1/${path}`);
  target.search = request.nextUrl.search;

  const token = await getSessionToken();
  const headers: Record<string, string> = { Accept: "application/json", ...visitorHeaders(request.headers) };
  if (token) headers.Authorization = `Bearer ${token}`;
  copyHeader(request.headers, headers, "content-type", "Content-Type");
  copyHeader(request.headers, headers, "x-request-id", "X-Request-ID");

  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method: request.method,
      headers,
      body: SAFE_METHODS.has(request.method) ? undefined : await request.text(),
      cache: "no-store",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
  } catch {
    return NextResponse.json(errorBody("network_error", NETWORK_ERROR_MESSAGE), { status: 502 });
  }

  const res = new NextResponse((await upstream.text()) || null, { status: upstream.status });
  res.headers.set("Cache-Control", "no-store");
  for (const name of ["content-type", "x-request-id", "retry-after"]) {
    const value = upstream.headers.get(name);
    if (value) res.headers.set(name, value);
  }
  return res;
}

/**
 * Rebuilds the upstream path from the route's segments, or returns null if
 * any segment could climb out of /v1. Next decodes each segment, so
 * "/api/v1/x%2F..%2F..%2Freadyz" arrives as one segment "x/../../readyz";
 * joined naively, the URL parser would resolve it to the API's /readyz.
 * Rejecting separators and dot segments, then re-encoding, keeps every
 * request inside /v1.
 */
function upstreamPath(segments: string[]): string | null {
  for (const s of segments) {
    if (s === "" || s === "." || s === ".." || s.includes("/") || s.includes("\\")) return null;
  }
  return segments.map(encodeURIComponent).join("/");
}

/**
 * Cross-site request forgery guard for state-changing calls, the same rule
 * Go's http.CrossOriginProtection uses: trust the browser's Sec-Fetch-Site
 * when present, else compare Origin with Host. Requests with neither header
 * come from non-browser clients, which can't ride a victim's cookie anyway.
 * SameSite=Lax on the session cookie is the first line of defence; this is
 * the second, and it also covers same-site sibling subdomains.
 */
function isCrossOrigin(request: NextRequest): boolean {
  const site = request.headers.get("sec-fetch-site");
  if (site) return site !== "same-origin" && site !== "none";

  const origin = request.headers.get("origin");
  if (!origin) return false;
  try {
    const host = request.headers.get("x-forwarded-host") ?? request.headers.get("host");
    return new URL(origin).host !== host;
  } catch {
    return true;
  }
}

function copyHeader(from: Headers, to: Record<string, string>, name: string, as: string) {
  const value = from.get(name);
  if (value) to[as] = value;
}

export { forward as GET, forward as POST, forward as PATCH, forward as PUT, forward as DELETE };
