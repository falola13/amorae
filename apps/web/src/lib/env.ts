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

function readCookieSecure(): boolean {
  if (process.env.COOKIE_SECURE !== undefined) {
    return process.env.COOKIE_SECURE === "true";
  }
  return process.env.NODE_ENV === "production";
}

export const env = {
  get API_URL(): string {
    return readApiUrl();
  },
  get COOKIE_SECURE(): boolean {
    return readCookieSecure();
  },
};
