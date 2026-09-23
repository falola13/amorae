import "server-only";

import { apiFetch } from "@/lib/api/client";
import type { RegisterInput } from "@/lib/api/schemas";
import type { AuthResult } from "@/lib/api/types";

export function login(email: string, password: string): Promise<AuthResult> {
  return apiFetch<AuthResult>("/auth/login", {
    method: "POST",
    body: { email, password },
  });
}

export function register(input: RegisterInput): Promise<AuthResult> {
  return apiFetch<AuthResult>("/auth/register", { method: "POST", body: input });
}

export function logout(token: string): Promise<void> {
  return apiFetch<void>("/auth/logout", { method: "POST", token });
}
