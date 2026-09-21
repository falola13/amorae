"use server";

import { redirect } from "next/navigation";

import { clearSession, getSessionToken, setSession } from "@/lib/auth/session";
import { toFormState, type FormState } from "@/lib/forms";
import { safeNext } from "@/lib/safe-next";
import { login, logout, register } from "./api";

export async function loginAction(
  _prevState: FormState,
  formData: FormData,
): Promise<FormState> {
  const email = String(formData.get("email") ?? "");
  const password = String(formData.get("password") ?? "");
  const next = safeNext(formData.get("next"));

  let result;
  try {
    result = await login(email, password);
  } catch (error) {
    return toFormState(error, { email });
  }

  // redirect() throws, so it must run after the try/catch — inside it, the
  // throw would be caught and reported as a form error instead of
  // navigating.
  await setSession(result.token, result.expires_at);
  redirect(next);
}

export async function registerAction(
  _prevState: FormState,
  formData: FormData,
): Promise<FormState> {
  const email = String(formData.get("email") ?? "");
  const password = String(formData.get("password") ?? "");
  const displayName = String(formData.get("display_name") ?? "");

  let result;
  try {
    result = await register(email, password, displayName);
  } catch (error) {
    return toFormState(error, { email, display_name: displayName });
  }

  await setSession(result.token, result.expires_at);
  redirect("/dashboard");
}

export async function logoutAction(): Promise<void> {
  const token = await getSessionToken();
  if (token) {
    try {
      await logout(token);
    } catch {
      // The cookie is cleared below regardless of whether the API call
      // succeeded — a failed logout request must never strand the user in
      // a logged-in-looking state.
    }
  }
  await clearSession();
  redirect("/login");
}
