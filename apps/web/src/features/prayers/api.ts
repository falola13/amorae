import { apiPath, http } from "@/lib/api/http";
import type { PrayerPoint, PrayerWeek } from "@/lib/api/types";

export const prayersApi = {
  current: () => http.get<PrayerWeek>("/prayers/current").then((r) => r.data),
  history: () => http.get<PrayerWeek[]>("/prayers/history").then((r) => r.data),
  week: (id: string) => http.get<PrayerWeek>(apiPath`/prayers/weeks/${id}`).then((r) => r.data),
  savePoints: (points: PrayerPoint[]) => http.put<PrayerWeek>("/prayers/current/points", { points }).then((r) => r.data),
  publish: () => http.post<PrayerWeek>("/prayers/current/publish").then((r) => r.data),
  complete: (pointId: string) => http.post<PrayerWeek>(apiPath`/prayers/points/${pointId}/complete`).then((r) => r.data),
  uncomplete: (pointId: string) => http.delete<PrayerWeek>(apiPath`/prayers/points/${pointId}/complete`).then((r) => r.data),
  reflection: (weekId: string, reflection: string) => http.patch<PrayerWeek>(apiPath`/prayers/weeks/${weekId}/reflection`, { reflection }).then((r) => r.data),
  demo: (mode: "partner" | "mine" | "waiting") => http.post<PrayerWeek>("/prayers/current/demo", { mode }).then((r) => r.data),
};
