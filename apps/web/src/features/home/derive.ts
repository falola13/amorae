import { sumOf } from "@/features/together/goals";
import type { Event, Goal, NotificationPrefs, PrayerWeek } from "@/lib/api/types";
import { isApiError } from "@/lib/api/errors";
import { eventPhase } from "@/features/together/events";
import { time12 } from "@/lib/dates";

// True if no partner, or the API's 409 waiting_for_partner (only returned once a week was asked for).
export const isAlone = (hasPartner: boolean, weekError: unknown): boolean =>
  !hasPartner || (isApiError(weekError) && weekError.code === "waiting_for_partner");

/** Events still coming up or happening now, earliest first. */
export const upcomingEvents = (events: Event[], now: Date): Event[] =>
  events
    .filter((e) => eventPhase(e, now) !== "over")
    .sort((a, b) => (a.date + (a.start_time ?? "")).localeCompare(b.date + (b.start_time ?? "")));

/** Today's event: one on today's date, or an overnight one from yesterday that's still going. */
export const todayEvent = (upcoming: Event[], todayIso: string, now?: Date): Event | undefined =>
  upcoming.find((e) => e.date === todayIso || (now && eventPhase(e, now) === "ongoing"));

export const nextUpcomingEvent = (
  upcoming: Event[],
  todayIso: string,
  now?: Date,
): Event | undefined =>
  upcoming.find((e) => e.date !== todayIso && !(now && eventPhase(e, now) === "ongoing"));

/** The one goal still in progress, with its running total. */
export const activeGoal = (goals: Goal[]): { goal: Goal; total: number } | null => {
  const goal = goals.find((g) => !g.done);
  return goal ? { goal, total: sumOf(goal) } : null;
};

/** How many "This week" rows will show, for the "N things planned" caption. */
export const plannedCount = (
  upcoming: Event[],
  goal: { goal: Goal; total: number } | null,
  week: PrayerWeek | undefined,
): number => upcoming.length + (goal ? 1 : 0) + (week && week.status === "published" ? 1 : 0);

export type LastWeekSummary = { prayed: number; total: number; iSetIt: boolean };

/** The most recently finished week (history is newest-first and excludes the current week). */
export const lastWeekSummary = (
  history: PrayerWeek[] | undefined,
  meId: string,
): LastWeekSummary | null => {
  const w = history?.[0];
  return w
    ? { prayed: w.my_completed.length, total: w.points.length, iSetIt: w.setter_id === meId }
    : null;
};

/** "Off", or the reminder time as "7:00 pm". */
export const reminderLabel = (prefs: NotificationPrefs | undefined): string =>
  prefs?.prayer_reminder ? time12(prefs.reminder_time) : "Off";
