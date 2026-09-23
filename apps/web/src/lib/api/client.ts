import "server-only";

import { headers as requestHeaders } from "next/headers";

import { env } from "@/lib/env";
import { networkError, toApiError, unwrapData } from "./envelope";
import { visitorHeaders } from "./upstream";

// The server's HTTP client for the Go API, used by Server Actions (auth).
// Browser code uses lib/api/http.ts, which reaches the API through the
// /api/v1 proxy instead. Both share envelope.ts and upstream.ts.

interface ApiFetchOptions {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  token?: string;
}

const TIMEOUT_MS = 10_000;

// Feature modules pass a resource path ("/auth/login"). The version prefix
// lives here so a new API version is one change, not an edit to every caller.
const API_VERSION_PATH = "/v1";

export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const { method = "GET", body, token } = options;

  const headers: Record<string, string> = {
    Accept: "application/json",
    ...(await currentVisitorHeaders()),
  };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;

  let response: Response;
  try {
    response = await fetch(`${env.API_URL}${API_VERSION_PATH}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      // Every call carries a session or drives a mutation: never cache.
      cache: "no-store",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
  } catch {
    throw networkError();
  }

  if (response.status === 204) return undefined as T;

  const retryAfter = response.headers.get("retry-after");

  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    // A failure with no JSON body is the API's router talking, not a handler:
    // let toApiError read it (a bare 404 means "not built yet", not a crash).
    if (!response.ok) throw toApiError(response.status, undefined, retryAfter);
    throw networkError(response.status);
  }

  if (!response.ok) throw toApiError(response.status, payload, retryAfter);
  return unwrapData<T>(payload);
}

async function currentVisitorHeaders(): Promise<Record<string, string>> {
  try {
    return visitorHeaders(await requestHeaders());
  } catch {
    // Outside a request (e.g. at build time) there is no visitor.
    return {};
  }
}
