const FALLBACK = "/";
const BASE = "http://amorae.invalid";

// Only a same-origin path is a safe post-login redirect target. Resolving
// with the URL parser catches "\\evil.com" and friends the way browsers do.
export function safeNext(value: unknown): string {
  if (typeof value !== "string" || !value.startsWith("/")) return FALLBACK;
  try {
    const url = new URL(value, BASE);
    if (url.origin !== BASE) return FALLBACK;
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return FALLBACK;
  }
}
