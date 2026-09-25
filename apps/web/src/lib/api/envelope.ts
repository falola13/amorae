import { ApiError, isApiError } from "./errors";

// The Go API's response envelopes (docs/API.md), parsed in one place for both
// HTTP clients: lib/api/client.ts (server) and lib/api/http.ts (browser).

export const NETWORK_ERROR_MESSAGE = "We couldn't reach the server. Please try again.";
export const GENERIC_ERROR_MESSAGE = "Something went wrong. Please try again.";

/** The endpoint doesn't exist in the Go API yet, as opposed to a failure. */
export const NOT_AVAILABLE_CODE = "not_available";
export const NOT_AVAILABLE_MESSAGE = "This part of Amorae isn't available yet.";

/**
 * Codes that mean "there isn't one yet", not "something went wrong".
 *
 * A couple that has not formed, a week that cannot start until both people
 * are here, a module still being built. The request failed, but nothing is
 * broken — and the two deserve opposite answers on screen: an empty state,
 * or an error with a way to retry.
 */
const ABSENCE_CODES = new Set([NOT_AVAILABLE_CODE, "couple_not_found", "waiting_for_partner"]);

/**
 * Whether a failure is really an absence.
 *
 * Worth telling apart anywhere a failed request decides what somebody reads.
 * A brand-new account gets 404 couple_not_found from most of the API — it has
 * no couple yet — and reading that as breakage puts "this didn't load" across
 * a home screen that loaded perfectly well and simply has nothing in it.
 */
export function isAbsence(error: unknown): boolean {
  return isApiError(error) && (error.status === 404 || ABSENCE_CODES.has(error.code));
}

interface ErrorEnvelope {
  error?: { code?: string; message?: string; fields?: Record<string, string>; request_id?: string };
}

/** "in a moment" / "in 40 seconds" / "in about 2 minutes", for a 429's wait. */
export function waitPhrase(seconds: number): string {
  if (seconds <= 10) return "in a moment";
  if (seconds < 60) return `in ${seconds} seconds`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `in about ${minutes} ${minutes === 1 ? "minute" : "minutes"}`;
  const hours = Math.round(minutes / 60);
  return `in about ${hours} ${hours === 1 ? "hour" : "hours"}`;
}

/**
 * Maps a non-2xx response body to an ApiError whose message is safe to show.
 * `retryAfterHeader` is the response's `Retry-After`, which turns the API's
 * "wait a moment" into a wait the user can actually plan around.
 */
export function toApiError(status: number, body: unknown, retryAfterHeader?: unknown): ApiError {
  const err = (body as ErrorEnvelope | null | undefined)?.error;
  const retryAfter = Number(retryAfterHeader);
  if (status === 429 && Number.isFinite(retryAfter) && retryAfter > 0) {
    return new ApiError(
      status,
      err?.code ?? "rate_limited",
      `Too many attempts. Try again ${waitPhrase(Math.ceil(retryAfter))}.`,
      err?.fields,
      err?.request_id,
      Math.ceil(retryAfter),
    );
  }
  // The Go API wraps every error it sends, including "event not found", in
  // the envelope. A bare 404/405/501 comes from its router instead: nothing
  // is registered at that path or method yet. Retrying can't help, so say so.
  if (!err && (status === 404 || status === 405 || status === 501)) {
    return new ApiError(status, NOT_AVAILABLE_CODE, NOT_AVAILABLE_MESSAGE);
  }
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
