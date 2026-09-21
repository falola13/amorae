import "server-only";

import { headers as requestHeaders } from "next/headers";

import { env } from "@/lib/env";
import { ApiError } from "./errors";

interface ApiErrorEnvelope {
  error: {
    code: string;
    message: string;
    fields?: Record<string, string>;
    request_id?: string;
  };
}

interface ApiFetchOptions {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  token?: string;
}

const TIMEOUT_MS = 10_000;
const NETWORK_ERROR_MESSAGE = "We couldn't reach the server. Please try again.";

// Feature modules pass a resource path ("/auth/login"). The version prefix
// lives here so a new API version is one change, not an edit to every caller.
const API_VERSION_PATH = "/v1";

// The one place that talks to the Go API. Every feature's api.ts calls this
// instead of `fetch` directly, so the envelope unwrapping, auth header,
// timeout, and error mapping only exist once.
export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const { method = "GET", body, token } = options;

  const headers: Record<string, string> = { Accept: "application/json", ...(await visitorHeaders()) };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;

  let response: Response;
  try {
    response = await fetch(`${env.API_URL}${API_VERSION_PATH}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      // Every call here carries a user session or drives a mutation — none
      // of it should ever be served from Next's fetch cache.
      cache: "no-store",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
  } catch {
    throw new ApiError(0, "network_error", NETWORK_ERROR_MESSAGE);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    throw new ApiError(response.status, "network_error", NETWORK_ERROR_MESSAGE);
  }

  if (!response.ok) {
    const envelope = payload as Partial<ApiErrorEnvelope>;
    const err = envelope.error;
    throw new ApiError(
      response.status,
      err?.code ?? "unknown_error",
      err?.message ?? "Something went wrong. Please try again.",
      err?.fields,
      err?.request_id,
    );
  }

  return (payload as { data: T }).data;
}

// Every web request reaches the API from this server, so without these the
// API's per-IP rate limit would lump all visitors together. The API only
// believes X-Client-IP when X-BFF-Secret matches its own BFF_SECRET.
async function visitorHeaders(): Promise<Record<string, string>> {
  const secret = env.BFF_SECRET;
  if (!secret) return {};
  const ip = await visitorIP();
  return ip ? { "X-Client-IP": ip, "X-BFF-Secret": secret } : {};
}

// The rightmost X-Forwarded-For entry is the one written by the nearest hop:
// the reverse proxy in front of this server, or Next itself (from the socket)
// when nothing is in front. Earlier entries are whatever the client claimed.
// Deploy behind a proxy that sets the header (every host does); if Next faces
// the internet directly, a client-sent header survives and the per-IP limit
// can be dodged, though the API's per-account login limit still holds.
async function visitorIP(): Promise<string | undefined> {
  try {
    const incoming = await requestHeaders();
    const nearest = incoming.get("x-forwarded-for")?.split(",").at(-1)?.trim();
    return nearest || incoming.get("x-real-ip") || undefined;
  } catch {
    // Outside a request (e.g. at build time) there is no visitor.
    return undefined;
  }
}
