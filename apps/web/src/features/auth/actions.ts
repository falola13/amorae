"use server";

import { redirect } from "next/navigation";

import {
  loginSchema,
  registerSchema,
  type LoginInput,
  type RegisterInput,
} from "@/lib/api/schemas";
import { clearSession, getSessionToken, setSession } from "@/lib/auth/session";
import { toActionError, type ActionError } from "@/lib/forms";
import { safeNext } from "@/lib/safe-next";
import { login, logout, register } from "./api";
import { routes } from "@/lib/routes";

// Auth stays on Server Actions because only the server may set the httpOnly
// session cookie (docs/adr/0002). The forms validate with React Hook Form +
// Zod first; the same Zod schema runs again here.

export async function loginAction(input: LoginInput): Promise<ActionError | undefined> {
  const parsed = loginSchema.safeParse(input);
  if (!parsed.success)
    return { message: "Check the highlighted fields.", fields: zodFields(parsed.error) };
  const next = safeNext(parsed.data.next);

  let result;
  try {
    result = await login(parsed.data.email, parsed.data.password);
  } catch (error) {
    return toActionError(error);
  }
  await setSession(result.token, result.expires_at);
  redirect(next);
}

export async function registerAction(input: RegisterInput): Promise<ActionError | undefined> {
  const parsed = registerSchema.safeParse(input);
  if (!parsed.success)
    return { message: "Check the highlighted fields.", fields: zodFields(parsed.error) };

  let result;
  try {
    result = await register(parsed.data);
  } catch (error) {
    return toActionError(error);
  }
  await setSession(result.token, result.expires_at);
  redirect(routes.couple());
}

export async function logoutAction(): Promise<void> {
  const token = await getSessionToken();
  if (token) {
    try {
      await logout(token);
    } catch {
      // The cookie is cleared regardless: a failed logout call must never
      // strand the user in a logged-in-looking state.
    }
  }
  await clearSession();
  redirect(routes.welcome);
}

function zodFields(error: {
  issues: { path: PropertyKey[]; message: string }[];
}): Record<string, string> {
  const out: Record<string, string> = {};
  for (const i of error.issues) {
    const k = String(i.path[0] ?? "");
    if (k && !out[k]) out[k] = i.message;
  }
  return out;
}
