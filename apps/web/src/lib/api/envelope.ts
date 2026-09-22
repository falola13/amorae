import { ApiError } from "./errors";

// The Go API's response envelopes (docs/API.md), parsed in one place for both
// HTTP clients: lib/api/client.ts (server) and lib/api/http.ts (browser).

export const NETWORK_ERROR_MESSAGE = "We couldn't reach the server. Please try again.";
export const GENERIC_ERROR_MESSAGE = "Something went wrong. Please try again.";

interface ErrorEnvelope {
  error?: { code?: string; message?: string; fields?: Record<string, string>; request_id?: string };
}

/** Maps a non-2xx response body to an ApiError whose message is safe to show. */
export function toApiError(status: number, body: unknown): ApiError {
  const err = (body as ErrorEnvelope | null | undefined)?.error;
  return new ApiError(
    status,
    err?.code ?? "unknown_error",
    err?.message ?? GENERIC_ERROR_MESSAGE,
    err?.fields,
    err?.request_id,
  );
}

/** The request never produced a usable response (offline, timeout, bad gateway). */
export function networkError(status = 0): ApiError {
  return new ApiError(status, "network_error", NETWORK_ERROR_MESSAGE);
}

/** Unwraps `{ "data": T }`. */
export function unwrapData<T>(body: unknown): T {
  return (body as { data: T }).data;
}

/** The JSON body the web server itself returns when it has to fail a proxied call. */
export function errorBody(code: string, message: string) {
  return { error: { code, message } };
}
