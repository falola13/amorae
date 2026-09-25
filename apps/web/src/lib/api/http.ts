"use client";

import axios, { type AxiosError, type AxiosInstance } from "axios";

import { networkError, toApiError } from "./envelope";

/** Fired when the API says the session is gone; the app shell handles it. */
export const SESSION_EXPIRED_EVENT = "amorae:session-expired";

// Pages where a 401 is expected (signed out), so it must not bounce anyone.
const SIGNED_OUT_PATHS = /^\/(login|register|welcome)(\/|$)/;

// The browser's one HTTP client. Calls /api/v1/*; the route handler at
// src/app/api/v1/[...path]/route.ts adds the bearer token so it never reaches JS.
export const http: AxiosInstance = axios.create({
  baseURL: "/api/v1",
  timeout: 10_000,
  headers: { Accept: "application/json" },
});

/** apiPath`/events/${id}` — URL-encodes every interpolated value. Ids come from
 *  the page URL, so unencoded they could redirect a call to another endpoint. */
export function apiPath(strings: TemplateStringsArray, ...values: (string | number)[]): string {
  return strings.reduce(
    (out, s, i) => out + s + (i < values.length ? encodeURIComponent(String(values[i])) : ""),
    "",
  );
}

// Unwrap the `{ data }` envelope and map every failure to ApiError.
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
