import "server-only";

import { env } from "@/lib/env";

// Headers shared by both server-to-Go call sites (lib/api/client.ts and the
// browser proxy at app/api/v1/[...path]/route.ts) so they can't drift.

/** Without these the API's per-IP rate limit would lump all visitors together;
 *  it only trusts X-Client-IP when X-BFF-Secret matches its own BFF_SECRET. */
export function visitorHeaders(incoming: Headers): Record<string, string> {
  const headers: Record<string, string> = {};

  // So sessions show a real device in "where you're signed in" instead of this server's own agent.
  const userAgent = incoming.get("user-agent");
  if (userAgent) headers["User-Agent"] = userAgent;

  const secret = env.BFF_SECRET;
  if (!secret) return headers;
  const ip = visitorIP(incoming);
  if (ip) {
    headers["X-Client-IP"] = ip;
    headers["X-BFF-Secret"] = secret;
  }
  return headers;
}

/** Rightmost X-Forwarded-For entry = nearest hop; earlier ones are client-claimed.
 *  Must run behind a proxy that sets this header, or a spoofed one lets the
 *  per-IP limit be dodged (the per-account login limit still holds either way). */
function visitorIP(incoming: Headers): string | undefined {
  const nearest = incoming.get("x-forwarded-for")?.split(",").at(-1)?.trim();
  return nearest || incoming.get("x-real-ip") || undefined;
}
