import { daysUntil } from "@/lib/dates";

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

/** "7 years", "One year", or nothing at all the first time round. */
export const yearsBy = (date: string, day: string) => {
  const years = Number(day.slice(0, 4)) - Number(date.slice(0, 4));
  if (years <= 0) return "";
  return years === 1 ? "One year" : `${years} years`;
};
