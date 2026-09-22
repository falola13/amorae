import { http } from "@/lib/api/http";
import type { NotificationPrefs } from "@/lib/api/types";

export const settingsApi = {
  prefs: () => http.get<NotificationPrefs>("/notifications/preferences").then((r) => r.data),
  savePrefs: (patch: Partial<NotificationPrefs>) => http.patch<NotificationPrefs>("/notifications/preferences", patch).then((r) => r.data),
};
