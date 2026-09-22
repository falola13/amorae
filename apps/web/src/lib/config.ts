// Public, build-time configuration, safe to import from server and client
// code alike. NEXT_PUBLIC_* values are inlined when the app is built, so the
// server and browser bundles can never disagree about them.

/**
 * Mock mode: the browser's API client answers from a localStorage store and
 * the auth actions sign anyone in (see lib/env.ts for the production guard).
 * One flag drives both halves, so they can't drift apart.
 */
export const isMockApi = process.env.NEXT_PUBLIC_API_MOCK === "true";
