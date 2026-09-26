import { http } from "@/lib/api/http";
import type { NotificationsInbox } from "@/lib/api/types";

export const notificationsApi = {
  inbox: () => http.get<NotificationsInbox>("/notifications/inbox").then((r) => r.data),
  markAllRead: () => http.post("/notifications/inbox/read").then(() => undefined),
};
