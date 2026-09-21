import "server-only";

import { cookies } from "next/headers";

import { env } from "@/lib/env";

const COOKIE_NAME = "amorae_session";

export async function getSessionToken(): Promise<string | undefined> {
  const store = await cookies();
  return store.get(COOKIE_NAME)?.value;
}

// Called from Server Actions only — cookies() can't be mutated during
// Server Component rendering (see Next's cookies() docs).
export async function setSession(token: string, expiresAt: string): Promise<void> {
  const store = await cookies();
  store.set(COOKIE_NAME, token, {
    httpOnly: true,
    secure: env.COOKIE_SECURE,
    sameSite: "lax",
    path: "/",
    expires: new Date(expiresAt),
  });
}

export async function clearSession(): Promise<void> {
  const store = await cookies();
  store.delete(COOKIE_NAME);
}
