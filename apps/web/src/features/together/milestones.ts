import { daysUntil } from "@/lib/dates";

/**
 * A kept date recurs every year, so "next" means today if it hasn't happened
 * yet this year, otherwise the same day next year. Shared by the hub (soonest
 * upcoming date) and the milestones list (grouping).
 *
 * The twenty-ninth of February moves to the twenty-eighth in the three years
 * out of four that have no twenty-ninth — the same rule the server uses to
 * decide when to send the reminder (notifications.OccursOn), so the day the
 * screen shows and the day the notification arrives are never a day apart.
 */
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

/**
 * How far off a kept date is, in words: "Today", "Tomorrow", "in 12 days".
 *
 * It used to be `in ${daysUntil(...)} days` for every row, which on the one
 * day that matters most read "in 0 days" — and the day before it, "in 1
 * days". A countdown that cannot say today is not a countdown.
 */
export const countdown = (next: string, from: Date) => {
  const days = daysUntil(next, from);
  if (days <= 0) return "Today";
  if (days === 1) return "Tomorrow";
  return `in ${days} days`;
};
