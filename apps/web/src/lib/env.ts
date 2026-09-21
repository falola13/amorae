import "server-only";

// Read lazily (not at module load) so a misconfigured API_URL only breaks
// the request that actually needs it, with a clear error, instead of
// crashing the whole server at boot.
function readApiUrl(): string {
  const value = process.env.API_URL ?? "http://localhost:8088";
  try {
    new URL(value);
  } catch {
    throw new Error(`API_URL is not a valid URL: "${value}"`);
  }
  return value;
}

// Strict on purpose: this flag guards the session cookie. A lenient parse
// (`value === "true"`) would read COOKIE_SECURE=1 or TRUE as false and send
// the session over plain HTTP, so anything but true/false is an error.
function readCookieSecure(): boolean {
  const value = process.env.COOKIE_SECURE?.trim().toLowerCase();
  if (value === undefined || value === "") return process.env.NODE_ENV === "production";
  if (value === "true") return true;
  if (value === "false") return false;
  throw new Error(`COOKIE_SECURE must be "true" or "false", got "${process.env.COOKIE_SECURE}"`);
}

// Optional. Shared with the API so it believes the visitor IP this server
// forwards (see lib/api/client.ts). Must match the API's BFF_SECRET.
function readBffSecret(): string | undefined {
  const value = process.env.BFF_SECRET?.trim();
  if (!value) return undefined;
  if (value.length < 32) throw new Error("BFF_SECRET must be at least 32 characters when set");
  return value;
}

export const env = {
  get API_URL(): string {
    return readApiUrl();
  },
  get COOKIE_SECURE(): boolean {
    return readCookieSecure();
  },
  get BFF_SECRET(): string | undefined {
    return readBffSecret();
  },
};
