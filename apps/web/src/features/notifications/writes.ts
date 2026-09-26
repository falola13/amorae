import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { notificationsApi } from "./api";

// Every write the notifications feature makes, declared once (see lib/query/mutations.ts).
export const notificationsWrites = {
  markAllRead: defineWrite({
    mutationKey: ["notifications", "inbox", "read"],
    mutationFn: () => notificationsApi.markAllRead(),
    invalidates: [keys.inbox],
    idempotent: "marking everything read twice leaves everything read.",
  }),
};
