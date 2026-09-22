import "server-only";

import { env } from "@/lib/env";

// Headers every server-to-Go call carries, whichever door it went through:
// Server Actions (lib/api/client.ts) or the browser proxy
// (app/api/v1/[...path]/route.ts). Kept here so the two can't drift.

/**
 * Every web request reaches the API from this server, so without these the
 * API's per-IP rate limit would lump all visitors together. The API only
 * believes X-Client-IP when X-BFF-Secret matches its own BFF_SECRET.
 */
export function visitorHeaders(incoming: Headers): Record<string, string> {
  const secret = env.BFF_SECRET;
  if (!secret) return {};
  const ip = visitorIP(incoming);
  return ip ? { "X-Client-IP": ip, "X-BFF-Secret": secret } : {};
}

/**
 * The rightmost X-Forwarded-For entry is the one written by the nearest hop:
 * the reverse proxy in front of this server, or Next itself (from the socket)
 * when nothing is in front. Earlier entries are whatever the client claimed.
 * Deploy behind a proxy that sets the header; if Next faces the internet
 * directly, a client-sent header survives and the per-IP limit can be
 * dodged, though the API's per-account login limit still holds.
 */
function visitorIP(incoming: Headers): string | undefined {
  const nearest = incoming.get("x-forwarded-for")?.split(",").at(-1)?.trim();
  return nearest || incoming.get("x-real-ip") || undefined;
}
