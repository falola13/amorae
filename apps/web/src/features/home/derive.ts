import { sumOf } from "@/features/together/goals";
import type { Event, Goal, NotificationPrefs, PrayerWeek } from "@/lib/api/types";
import { isApiError } from "@/lib/api/errors";
import { stillAhead, time12 } from "@/lib/dates";

/**
 * Whether this account is still one person.
 *
 * There are two ways of learning the same thing and the screen needs both.
 * The API says it plainly — 409 waiting_for_partner — when a week is asked
 * for. But the week is never asked for until a partner exists, because the
 * query is disabled until then, so a lone account produces no error to read.
 * A screen that waited for one sat on its loading skeleton forever, which is
 * exactly what the first person to sign up saw: a greeting, the words "This
 * week", and a grey bar that never resolved into anything.
 */
export const isAlone = (hasPartner: boolean, weekError: unknown): boolean =>
  !hasPartner || (isApiError(weekError) && weekError.code === "waiting_for_partner");

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
