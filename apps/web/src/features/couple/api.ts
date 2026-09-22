import { http } from "@/lib/api/http";
import type { ChangeEmailInput } from "@/lib/api/schemas";
import type { Couple, User } from "@/lib/api/types";

export const coupleApi = {
  me: () => http.get<User>("/users/me").then((r) => r.data),
  // Email is not patchable here: changing it needs the current password (changeEmail).
  updateMe: (patch: Partial<Pick<User, "display_name" | "timezone">>) => http.patch<User>("/users/me", patch).then((r) => r.data),
  changeEmail: (input: ChangeEmailInput) => http.put<User>("/users/me/email", input).then((r) => r.data),
  deleteMe: () => http.delete("/users/me").then(() => undefined),
  couple: () => http.get<Couple>("/couples/me").then((r) => r.data),
  create: () => http.post<Couple>("/couples", {}).then((r) => r.data),
  join: (code: string) => http.post<Couple>("/couples/join", { code }).then((r) => r.data),
  onboarding: (patch: Partial<Couple["onboarding"]>) => http.patch<Couple>("/couples/me/onboarding", patch).then((r) => r.data),
};
