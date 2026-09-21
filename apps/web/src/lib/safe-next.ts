const FALLBACK = "/dashboard";
const BASE = "http://amorae.invalid";

// Only a same-origin path is a safe post-login redirect target. Prefix checks
// alone are not enough: browsers normalise "\" to "/" and strip tabs and
// newlines, so "/\\evil.com" or "/\t/evil.com" become the off-site
// "//evil.com". Resolving with the URL parser (which applies the same rules)
// and checking the origin catches every variant.
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
