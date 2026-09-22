import "server-only";

import { isMockApi } from "@/lib/config";

// Read lazily so a misconfigured value only breaks the request that needs it.
function readApiUrl(): string {
  const value = process.env.API_URL ?? "http://localhost:8088";
  try {
    new URL(value);
  } catch {
    throw new Error(`API_URL is not a valid URL: "${value}"`);
  }
  return value;
}

// Strict on purpose: these flags guard security settings, and a lenient parse
// would read COOKIE_SECURE=1 or TRUE as false.
function readBool(name: string, fallback: boolean): boolean {
  const value = process.env[name]?.trim().toLowerCase();
  if (value === undefined || value === "") return fallback;
  if (value === "true") return true;
  if (value === "false") return false;
  throw new Error(`${name} must be "true" or "false", got "${process.env[name]}"`);
}

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
    return readBool("COOKIE_SECURE", process.env.NODE_ENV === "production");
  },
  get BFF_SECRET(): string | undefined {
    return readBffSecret();
  },
  /**
   * Whether the auth actions should issue a mock session. Mock sign-in
   * accepts any email and password, so a production build refuses it unless
   * ALLOW_MOCK_AUTH=true says the deployment is deliberately a demo.
   */
  get MOCK_AUTH(): boolean {
    if (!isMockApi) return false;
    if (process.env.NODE_ENV === "production" && !readBool("ALLOW_MOCK_AUTH", false)) {
      throw new Error(
        "NEXT_PUBLIC_API_MOCK is on in a production build. Mock sign-in accepts any password; " +
          "set ALLOW_MOCK_AUTH=true only for a deliberate public demo.",
      );
    }
    return true;
  },
};
