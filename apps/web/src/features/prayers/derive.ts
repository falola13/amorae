import type { PrayerWeek } from "@/lib/api/types";

/** "1 prayer" / "3 prayers" / "no prayers". */
export const prayerCount = (n: number): string =>
  n === 0 ? "no prayers" : n === 1 ? "1 prayer" : `${n} prayers`;

/** "you", or the partner's name — lowercase, for inline sentences like "set by {word}". */
export const setterWord = (
  week: PrayerWeek,
  meId: string | undefined,
  partnerName: string,
): string => (week.setter_id === meId ? "you" : partnerName);

/** "You", or the partner's name — capitalized, as the subject of a sentence. */
export const setterLabel = (
  week: PrayerWeek,
  meId: string | undefined,
  partnerName: string,
): string => (week.setter_id === meId ? "You" : partnerName);

/** "You set the prayers" / "{partner} set the prayers". */
export const setterSentence = (
  week: PrayerWeek,
  meId: string | undefined,
  partnerName: string,
): string => `${setterLabel(week, meId, partnerName)} set the prayers`;

/**
 * The prayer index to open prayer mode at: `at` if it's a whole number in
 * range, else the first prayer not yet marked prayed (or 0 if all are done).
 * Guards against `?at=` being missing, non-numeric, or out of bounds — any
 * of which used to reach `points[NaN]` and crash.
 */
export const startIndex = (week: PrayerWeek, at: string | null): number => {
  const n = at === null ? NaN : Number(at);
  if (Number.isInteger(n) && n >= 0 && n < week.points.length) return n;
  const first = week.points.findIndex((p) => !week.my_completed.includes(p.id));
  return first === -1 ? 0 : first;
};
