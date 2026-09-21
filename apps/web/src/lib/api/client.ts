import "server-only";

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

// The one place that talks to the Go API. Every feature's api.ts calls this
// instead of `fetch` directly, so the envelope unwrapping, auth header,
// timeout, and error mapping only exist once.
export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const { method = "GET", body, token } = options;

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;

  let response: Response;
  try {
    response = await fetch(`${env.API_URL}${path}`, {
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
