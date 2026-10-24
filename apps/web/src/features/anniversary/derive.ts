import type { Anniversary, AnniversaryYear, Appreciation, Memory } from "@/lib/api/types";
import { daysUntil, parse } from "@/lib/dates";

const WORDS = ["", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten"];

/** "Three years", "One year", "12 years". */
export const yearsWord = (n: number) => `${WORDS[n] ?? n} ${n === 1 ? "year" : "years"}`;

/** 1st, 2nd, 3rd, 4th … 11th, 12th, 13th, 21st. */
export const ordinal = (n: number) => {
  const tens = n % 100;
  const suffix = tens >= 11 && tens <= 13 ? "th" : (["th", "st", "nd", "rd"][n % 10] ?? "th");
  return `${n}${suffix}`;
};

/** The anniversary is today. */
export const isTheDay = (a: Anniversary | undefined) => a?.days_until === 0;

/** "today", "tomorrow", "in 5 days" — for the days before. */
export const whenLabel = (days: number) =>
  days <= 0 ? "today" : days === 1 ? "tomorrow" : `in ${days} days`;

/** Days from the day they started to `todayIso`, counting neither end twice. */
export const daysTogether = (startedOn: string, todayIso: string) =>
  Math.max(0, daysUntil(todayIso, parse(startedOn)));

/** 1096 → "1,096". */
export const count = (n: number) => n.toLocaleString("en-GB");

export interface Tile {
  n: number;
  label: string;
}

/** The year in numbers, the ones worth saying — nothing at zero, which
 *  would read as a reproach on a day that should not have one. */
export function yearTiles(y: AnniversaryYear): Tile[] {
  const tiles: Tile[] = [
    { n: y.prayed, label: y.prayed === 1 ? "prayer prayed" : "prayers prayed" },
    { n: y.prayer_weeks, label: y.prayer_weeks === 1 ? "week of prayer" : "weeks of prayer" },
    { n: y.answered, label: y.answered === 1 ? "prayer answered" : "prayers answered" },
    { n: y.memories, label: y.memories === 1 ? "memory kept" : "memories kept" },
    { n: y.events, label: y.events === 1 ? "plan that happened" : "plans that happened" },
    { n: y.challenges, label: y.challenges === 1 ? "challenge finished" : "challenges finished" },
    { n: y.notes, label: y.notes === 1 ? "note to each other" : "notes to each other" },
    { n: y.journal, label: y.journal === 1 ? "journal entry" : "journal entries" },
  ];
  return tiles.filter((t) => t.n > 0);
}

const inYear = (date: string, y: AnniversaryYear) => date >= y.from && date < y.to;

/** Up to `max` notes from the year, spread across it rather than the latest
 *  few, oldest first so they read in the order they were sent. */
export function notesOfTheYear(notes: Appreciation[], y: AnniversaryYear, max = 3): Appreciation[] {
  const within = notes
    .filter((n) => inYear(n.date, y))
    .sort((a, b) => a.date.localeCompare(b.date));
  return spread(within, max);
}

/** Up to `max` photographed memories from the year, spread across it, oldest first. */
export function photosOfTheYear(memories: Memory[], y: AnniversaryYear, max = 6): Memory[] {
  const within = memories
    .filter((m) => m.photo_url && inYear(m.date, y))
    .sort((a, b) => a.date.localeCompare(b.date));
  return spread(within, max);
}

function spread<T>(items: T[], max: number): T[] {
  if (items.length <= max) return items;
  if (max <= 1) return items.slice(0, max);
  // First and last always, the rest evenly between: the whole year, not its start.
  const step = (items.length - 1) / (max - 1);
  return Array.from({ length: max }, (_, i) => items[Math.round(i * step)]);
}

/** Whether this device has already shown the opening for that day. */
const seenKey = (on: string) => `amorae.anniversary.opened.${on}`;

export function openingSeen(on: string): boolean {
  try {
    return window.localStorage.getItem(seenKey(on)) === "1";
  } catch {
    return false;
  }
}

export function markOpeningSeen(on: string) {
  try {
    window.localStorage.setItem(seenKey(on), "1");
  } catch {
    // Private mode: it shows again next time, which is no harm on this day.
  }
}
