const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];
const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export const parse = (iso: string) => {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d);
};
export const iso = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
export const addDays = (d: Date, n: number) => {
  const x = new Date(d);
  x.setDate(x.getDate() + n);
  return x;
};
export const startOfWeek = (d: Date) => addDays(d, -d.getDay()); // Sunday

export const dayName = (isoDate: string) => DAYS[parse(isoDate).getDay()];
export const monthName = (isoDate: string) => MONTHS[parse(isoDate).getMonth()];
export const shortMonth = (isoDate: string) => MONTHS[parse(isoDate).getMonth()].slice(0, 3);
export const dayNum = (isoDate: string) => parse(isoDate).getDate();

/** "27 September" */
export const longDate = (isoDate: string) => `${dayNum(isoDate)} ${monthName(isoDate)}`;
/** "Tuesday 29 September" */
export const weekdayDate = (isoDate: string) => `${dayName(isoDate)} ${longDate(isoDate)}`;
/** "27 Sep - 3 Oct" or "27 September - 3 October" */
export const range = (a: string, b: string, short = false) => {
  const m = short ? shortMonth : monthName;
  return m(a) === m(b)
    ? `${dayNum(a)}-${dayNum(b)} ${m(a)}`
    : `${dayNum(a)} ${m(a)} - ${dayNum(b)} ${m(b)}`;
};
export const daysUntil = (isoDate: string, from = new Date()) =>
  Math.round((parse(isoDate).getTime() - parse(iso(from)).getTime()) / 86400000);

/** "7:00 pm" from "19:00" */
export const time12 = (t?: string) => {
  if (!t) return "";
  const [h, m] = t.split(":").map(Number);
  return `${((h + 11) % 12) + 1}:${String(m).padStart(2, "0")} ${h < 12 ? "am" : "pm"}`;
};

export const greeting = (d = new Date()) => {
  const h = d.getHours();
  return h < 12 ? "Good morning" : h < 17 ? "Good afternoon" : "Good evening";
};
export const relativeDay = (isoDate: string, today: string) => {
  const n = daysUntil(isoDate, parse(today));
  if (n === 0) return "Today";
  if (n === 1) return "Tomorrow";
  if (n > 1 && n < 7) return dayName(isoDate);
  return longDate(isoDate);
};

export const naira = (n: number) => `₦${n.toLocaleString("en-NG")}`;

/**
 * How long ago a timestamp was, in words: "just now", "2 hours ago",
 * "3 days ago", then the date once it stops being useful as a duration.
 * For session lists, where "when" only has to be recognisable.
 */
export const timeAgo = (isoTimestamp: string, now = new Date()) => {
  const then = new Date(isoTimestamp);
  const minutes = Math.round((now.getTime() - then.getTime()) / 60000);
  if (minutes < 2) return "just now";
  if (minutes < 60) return `${minutes} minutes ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours} ${hours === 1 ? "hour" : "hours"} ago`;
  const days = Math.round(hours / 24);
  if (days < 7) return `${days} ${days === 1 ? "day" : "days"} ago`;
  return longDate(iso(then));
};
