import { NextResponse, type NextRequest } from "next/server";

import { errorBody, NETWORK_ERROR_MESSAGE } from "@/lib/api/envelope";
import { visitorHeaders } from "@/lib/api/upstream";
import { getSessionToken } from "@/lib/auth/session";
import { env } from "@/lib/env";

// Same-origin proxy to <API_URL>/v1/<path>, attaching the session bearer token; the browser never holds it (ADR-0002).

// Pinned to Frankfurt: API and DB are both there, and every call is uncached, so a mismatched region costs a round-trip on every request.
export const preferredRegion = "fra1";

const TIMEOUT_MS = 10_000;
const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS"]);

async function forward(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  if (!SAFE_METHODS.has(request.method) && isCrossOrigin(request)) {
    return NextResponse.json(errorBody("forbidden", "This request came from another site."), {
      status: 403,
    });
  }

  const path = upstreamPath((await params).path);
  if (path === null) {
    return NextResponse.json(errorBody("route_not_found", "That page doesn't exist."), {
      status: 404,
    });
  }

  const target = new URL(`${env.API_URL}/v1/${path}`);
  target.search = request.nextUrl.search;

  const token = await getSessionToken();
  const headers: Record<string, string> = {
    Accept: "application/json",
    ...visitorHeaders(request.headers),
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  copyHeader(request.headers, headers, "content-type", "Content-Type");
  copyHeader(request.headers, headers, "x-request-id", "X-Request-ID");
  // Without this the key never reaches the API and the whole thing is a no-op
  // that passes its own unit tests (FR-PWA-009).
  copyHeader(request.headers, headers, "idempotency-key", "Idempotency-Key");

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
  for (const name of [
    "content-type",
    "x-request-id",
    "retry-after",
    "content-disposition",
    // So a caller can tell a replayed reply from the original.
    "idempotent-replay",
  ]) {
    const value = upstream.headers.get(name);
    if (value) res.headers.set(name, value);
  }
  return res;
}

// Rejects segments with separators or dot-segments (Next decodes each one, so an
// encoded "../" could otherwise escape /v1) and re-encodes the rest.
function upstreamPath(segments: string[]): string | null {
  for (const s of segments) {
    if (s === "" || s === "." || s === ".." || s.includes("/") || s.includes("\\")) return null;
  }
  return segments.map(encodeURIComponent).join("/");
}

// CSRF guard for state-changing calls: trusts Sec-Fetch-Site when present, else compares Origin to Host.
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
