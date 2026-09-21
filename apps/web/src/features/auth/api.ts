import "server-only";

import { apiFetch } from "@/lib/api/client";
import type { AuthResult } from "@/lib/api/types";

export function login(email: string, password: string): Promise<AuthResult> {
  return apiFetch<AuthResult>("/v1/auth/login", {
    method: "POST",
    body: { email, password },
  });
}

export function register(email: string, password: string, displayName: string): Promise<AuthResult> {
  return apiFetch<AuthResult>("/v1/auth/register", {
    method: "POST",
    body: { email, password, display_name: displayName },
  });
}

export function logout(token: string): Promise<void> {
  return apiFetch<void>("/v1/auth/logout", { method: "POST", token });
}
