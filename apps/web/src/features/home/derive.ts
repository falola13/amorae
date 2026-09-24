import { sumOf } from "@/features/together/goals";
import type { Event, Goal, NotificationPrefs, PrayerWeek } from "@/lib/api/types";
import { stillAhead, time12 } from "@/lib/dates";

/** Events not yet done and not yet over, earliest first. */
export const upcomingEvents = (events: Event[], now: Date): Event[] =>
  events
    .filter((e) => !e.done && stillAhead(e.date, e.start_time, now))
    .sort((a, b) => (a.date + (a.start_time ?? "")).localeCompare(b.date + (b.start_time ?? "")));

/**
 * The first thing happening today. It used to be called tonightEvent, which
 * was only ever true after five o'clock — see partOfDay for what the card
 * now calls it.
 */
export const todayEvent = (upcoming: Event[], todayIso: string): Event | undefined =>
  upcoming.find((e) => e.date === todayIso);

export const nextUpcomingEvent = (upcoming: Event[], todayIso: string): Event | undefined =>
  upcoming.find((e) => e.date !== todayIso);

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
