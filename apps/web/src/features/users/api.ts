import "server-only";

import { apiFetch } from "@/lib/api/client";
import type { User } from "@/lib/api/types";
import { getSessionToken } from "@/lib/auth/session";

export async function getCurrentUser(): Promise<User> {
  const token = await getSessionToken();
  return apiFetch<User>("/users/me", { token });
}

export async function updateProfile(displayName: string): Promise<User> {
  const token = await getSessionToken();
  return apiFetch<User>("/users/me", {
    method: "PATCH",
    body: { display_name: displayName },
    token,
  });
}
