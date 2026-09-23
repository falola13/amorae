import type { NotificationPrefs } from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { settingsApi } from "./api";

// Every write the settings feature makes, declared once (see lib/query/mutations.ts).
export const settingsWrites = {
  // Switches flip in order, so quick on-off-on taps can't land out of order.
  savePrefs: defineWrite({
    mutationKey: ["notifications", "preferences"],
    mutationFn: (patch: Partial<NotificationPrefs>) => settingsApi.savePrefs(patch),
    invalidates: [keys.prefs],
    scope: "settings.prefs",
  }),
  // Securing an account is only meaningful now: queued and replayed an hour
  // later it would sign out devices the person has since decided to keep.
  signOutOthers: defineWrite({
    mutationKey: ["sessions", "sign-out-others"],
    mutationFn: () => settingsApi.signOutOthers(),
    invalidates: [keys.sessions],
    onlineOnly: true,
  }),
};
