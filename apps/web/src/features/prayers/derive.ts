import type { PrayerPoint, PrayerWeek } from "@/lib/api/types";
import { parse } from "@/lib/dates";

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

// Local weekday (0=Sun..6=Sat) for an ISO date. Parsed with lib/dates' `parse`, which builds a
// local `Date` from the y/m/d parts — never `new Date(iso)`, which UTC-parses and can land on
// the wrong day depending on the reader's timezone.
export const weekdayOf = (isoDate: string): number => parse(isoDate).getDay();

/** A point is "for" a weekday if it has no chosen days (every day) or that day is one of them. */
export const isForDay = (point: PrayerPoint, weekday: number): boolean =>
  point.weekdays.length === 0 || point.weekdays.includes(weekday);

/**
 * This week's points scheduled for today. Empty when the week carries no `today` — which is
 * exactly the case for a week from history, since the API only sends it for the current one.
 */
export const todaysPoints = (week: PrayerWeek): PrayerPoint[] => {
  if (!week.today) return [];
  const weekday = weekdayOf(week.today);
  return week.points.filter((p) => isForDay(p, weekday));
};

const SHORT_DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

/** "Every day", or the chosen days short and in order: "Mon · Thu". */
export const daysLabel = (weekdays: number[]): string =>
  weekdays.length === 0
    ? "Every day"
    : [...weekdays]
        .sort((a, b) => a - b)
        .map((d) => SHORT_DAYS[d])
        .join(" · ");

// Index to open prayer mode at, within TODAY's points: `at` if valid, else the first of today's
// points not yet prayed (or 0). Guards against `?at=` being missing, non-numeric, or out of bounds.
export const startIndex = (week: PrayerWeek, at: string | null): number => {
  const todays = todaysPoints(week);
  const n = at === null ? NaN : Number(at);
  if (Number.isInteger(n) && n >= 0 && n < todays.length) return n;
  const first = todays.findIndex((p) => !week.my_completed.includes(p.id));
  return first === -1 ? 0 : first;
};
