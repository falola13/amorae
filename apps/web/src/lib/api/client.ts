import "server-only";

import { headers as requestHeaders } from "next/headers";

import { env } from "@/lib/env";
import { networkError, toApiError, unwrapData } from "./envelope";
import { visitorHeaders } from "./upstream";

// Server HTTP client for the Go API (Server Actions/auth). Browser code uses
// lib/api/http.ts through the /api/v1 proxy instead; both share envelope.ts and upstream.ts.

interface ApiFetchOptions {
  method?: "GET" | "POST" | "PATCH" | "DELETE";
  body?: unknown;
  token?: string;
}

const TIMEOUT_MS = 10_000;

// Version prefix lives here so a new API version is one change, not every caller.
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
    // No JSON body means the router is talking, not a handler (e.g. a bare 404).
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
