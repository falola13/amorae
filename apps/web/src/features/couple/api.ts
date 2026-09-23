import { http } from "@/lib/api/http";
import type { ChangeEmailInput, ChangePasswordInput, DeleteAccountInput } from "@/lib/api/schemas";
import type { Couple, User } from "@/lib/api/types";

export const coupleApi = {
  me: () => http.get<User>("/users/me").then((r) => r.data),
  // Email is not patchable here: changing it needs the current password (changeEmail).
  updateMe: (patch: Partial<Pick<User, "display_name" | "timezone">>) =>
    http.patch<User>("/users/me", patch).then((r) => r.data),
  changeEmail: (input: ChangeEmailInput) =>
    http.put<User>("/users/me/email", input).then((r) => r.data),
  changePassword: (input: ChangePasswordInput) =>
    http.put("/users/me/password", input).then(() => undefined),
  // Axios puts a DELETE body in the config's `data`, not a second argument.
  deleteMe: (input: DeleteAccountInput) =>
    http.delete("/users/me", { data: input }).then(() => undefined),
  couple: () => http.get<Couple>("/couples/me").then((r) => r.data),
  create: () => http.post<Couple>("/couples", {}).then((r) => r.data),
  join: (code: string) => http.post<Couple>("/couples/join", { code }).then((r) => r.data),
  regenerateInvite: () => http.post<Couple>("/couples/invite", {}).then((r) => r.data),
  updateCouple: (patch: { name?: string; relationship_start_date?: string }) =>
    http.patch<Couple>("/couples/me", patch).then((r) => r.data),
  updateRole: (role: string) => http.patch<Couple>("/couples/role", { role }).then((r) => r.data),
  onboarding: (patch: Partial<Couple["onboarding"]>) =>
    http.patch<Couple>("/couples/me/onboarding", patch).then((r) => r.data),
  // Signed out: the BFF forwards these without a session.
  forgotPassword: (email: string) =>
    http.post("/auth/password/forgot", { email }).then(() => undefined),
  resetPassword: (token: string, new_password: string) =>
    http.post("/auth/password/reset", { token, new_password }).then(() => undefined),
};
