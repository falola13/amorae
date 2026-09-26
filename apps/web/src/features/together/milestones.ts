import { daysUntil, parse } from "@/lib/dates";
import type { Milestone } from "@/lib/api/types";

// "Next" = today if not yet passed this year, else next year. Feb 29 maps to Feb 28 in non-leap
// years — same rule as the server's notifications.OccursOn, so screen and reminder never disagree.
export const nextOccurrence = (date: string, todayIso: string) => {
  if (date >= todayIso) return date;
  const [, m, d] = date.split("-");
  const y = Number(todayIso.slice(0, 4));
  const thisYear = dayIn(y, m, d);
  return thisYear >= todayIso ? thisYear : dayIn(y + 1, m, d);
};

const isLeapYear = (y: number) => y % 4 === 0 && (y % 100 !== 0 || y % 400 === 0);

const dayIn = (year: number, month: string, day: string) => {
  const onTheDay = month === "02" && day === "29" && !isLeapYear(year) ? "28" : day;
  return `${year}-${month}-${onTheDay}`;
};

/** How far off a kept date is: "Today", "Tomorrow", "in 12 days". */
export const countdown = (next: string, from: Date) => {
  const days = daysUntil(next, from);
  if (days <= 0) return "Today";
  if (days === 1) return "Tomorrow";
  return `in ${days} days`;
};

// Same month/day, any year — used per calendar cell since "next" is relative to today and a paged
// calendar needs an answer that doesn't move. Feb 29 maps to Feb 28 in non-leap years, like the reminder.
export const occursOn = (date: string, day: string) => {
  const [, month, dayOfMonth] = date.split("-");
  return dayIn(Number(day.slice(0, 4)), month, dayOfMonth) === day;
};

/** "7 years", "One year", or nothing at all the first time round — or when
 *  there's no year to count from (a birthday with none on file: `date` then
 *  carries 2000 as a placeholder, which must never surface as an age). */
export const yearsBy = (date: string, day: string, yearKnown = true) => {
  if (!yearKnown) return "";
  const years = Number(day.slice(0, 4)) - Number(date.slice(0, 4));
  if (years <= 0) return "";
  return years === 1 ? "One year" : `${years} years`;
};

/** How many days until a date's next occurrence, from today — 0 when it's today. */
export const daysAway = (date: string, todayIso: string): number =>
  daysUntil(nextOccurrence(date, todayIso), parse(todayIso));

/** Every milestone whose next occurrence lands on today. */
export const todaysMilestones = (milestones: Milestone[], todayIso: string): Milestone[] =>
  milestones.filter((m) => occursOn(m.date, todayIso));

/** Milestones landing in the next `withinDays` days, not counting today — soonest first. */
export const upcomingMilestones = (
  milestones: Milestone[],
  todayIso: string,
  withinDays = 7,
): Array<Milestone & { days: number }> =>
  milestones
    .map((m) => ({ ...m, days: daysAway(m.date, todayIso) }))
    .filter((m) => m.days > 0 && m.days <= withinDays)
    .sort((a, b) => a.days - b.days);

/** "in 5 days" / "tomorrow" for the slim upcoming row; `days` must be > 0. */
export const daysAwayLabel = (days: number): string =>
  days === 1 ? "tomorrow" : `in ${days} days`;

/** The celebration card's title and sub for a milestone occurring today.
 *  `meId` and `partnerName` disambiguate whose birthday it is. */
export function celebrationCopy(
  m: Milestone,
  todayIso: string,
  meId: string | undefined,
  partnerName: string,
): { title: string; sub: string } {
  if (m.source === "birthday") {
    if (m.about === meId) {
      return { title: "Happy birthday", sub: `${partnerName} has been waiting for today.` };
    }
    return { title: `Happy birthday, ${partnerName}`, sub: "Tell them today." };
  }
  if (m.source === "anniversary") {
    const years = yearsBy(m.date, todayIso, true);
    return { title: "Happy anniversary", sub: years ? `${years} together` : "" };
  }
  return { title: m.title, sub: "Today" };
}
