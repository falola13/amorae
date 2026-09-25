import { apiPath, http } from "@/lib/api/http";
import type { AnsweredPrayer, PrayerPoint, PrayerWeek } from "@/lib/api/types";

export const prayersApi = {
  current: () => http.get<PrayerWeek>("/prayers/current").then((r) => r.data),
  history: () => http.get<PrayerWeek[]>("/prayers/history").then((r) => r.data),
  week: (id: string) => http.get<PrayerWeek>(apiPath`/prayers/weeks/${id}`).then((r) => r.data),
  savePoints: (points: PrayerPoint[]) =>
    http.put<PrayerWeek>("/prayers/current/points", { points }).then((r) => r.data),
  publish: () => http.post<PrayerWeek>("/prayers/current/publish").then((r) => r.data),
  complete: (pointId: string) =>
    http.post<PrayerWeek>(apiPath`/prayers/points/${pointId}/complete`).then((r) => r.data),
  uncomplete: (pointId: string) =>
    http.delete<PrayerWeek>(apiPath`/prayers/points/${pointId}/complete`).then((r) => r.data),
  answered: () => http.get<AnsweredPrayer[]>("/prayers/answered").then((r) => r.data),
  // PUT, not POST: saying this twice means the same as saying it once, and
  // the second time is an edit of the note rather than a second answer.
  setAnswered: (pointId: string, note: string) =>
    http
      .put<PrayerWeek>(apiPath`/prayers/points/${pointId}/answered`, { note })
      .then((r) => r.data),
  unsetAnswered: (pointId: string) =>
    http.delete<PrayerWeek>(apiPath`/prayers/points/${pointId}/answered`).then((r) => r.data),
  reflection: (weekId: string, reflection: string) =>
    http
      .patch<PrayerWeek>(apiPath`/prayers/weeks/${weekId}/reflection`, { reflection })
      .then((r) => r.data),
};
