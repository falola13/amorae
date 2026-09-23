"use client";

import axios, { type AxiosError, type AxiosInstance } from "axios";

import { networkError, toApiError } from "./envelope";

/** Fired when the API says the session is gone; the app shell handles it. */
export const SESSION_EXPIRED_EVENT = "amorae:session-expired";

// Pages where a 401 is expected (signed out), so it must not bounce anyone.
const SIGNED_OUT_PATHS = /^\/(login|register|welcome)(\/|$)/;

// The browser's one HTTP client. It calls /api/v1/* on this origin; the route
// handler at src/app/api/v1/[...path]/route.ts adds the session's bearer
// token and forwards to the Go API, so the token never reaches JavaScript
// (docs/adr/0002). Feature api.ts files call this, never axios directly.
export const http: AxiosInstance = axios.create({
  baseURL: "/api/v1",
  timeout: 10_000,
  headers: { Accept: "application/json" },
});

/**
 * Builds an API path with every interpolated value URL-encoded:
 * apiPath`/events/${id}`. Ids come from the page URL, so without encoding a
 * crafted link ("/together/events/..%2Fusers%2Fme") could point a call at a
 * different endpoint.
 */
export function apiPath(strings: TemplateStringsArray, ...values: (string | number)[]): string {
  return strings.reduce(
    (out, s, i) => out + s + (i < values.length ? encodeURIComponent(String(values[i])) : ""),
    "",
  );
}

// Unwrap the `{ data }` envelope and map every failure to ApiError, whose
// message is always safe to show.
http.interceptors.response.use(
  (response) => {
    if (response.status === 204 || response.data === undefined || response.data === "")
      return { ...response, data: undefined };
    return { ...response, data: (response.data as { data: unknown }).data };
  },
  (error: AxiosError) => {
    if (!error.response) throw networkError();
    const { status, data } = error.response;
    if (
      status === 401 &&
      typeof window !== "undefined" &&
      !SIGNED_OUT_PATHS.test(window.location.pathname)
    ) {
      // The app shell clears the query cache and sends the user to /login.
      window.dispatchEvent(new CustomEvent(SESSION_EXPIRED_EVENT));
    }
    throw toApiError(status, data, error.response.headers["retry-after"]);
  },
);
