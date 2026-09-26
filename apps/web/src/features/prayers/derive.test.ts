import { describe, expect, it } from "vitest";

import type { PrayerPoint, PrayerWeek } from "@/lib/api/types";
import { daysLabel, isForDay, startIndex, todaysPoints, weekdayOf } from "./derive";

const point = (over: Partial<PrayerPoint> = {}): PrayerPoint => ({
  id: "p1",
  title: "Peace",
  text: "",
  position: 0,
  weekdays: [],
  ...over,
});

const week = (over: Partial<PrayerWeek> = {}): PrayerWeek => ({
  id: "w1",
  week_start: "2026-09-20",
  week_end: "2026-09-26",
  setter_id: "me",
  status: "published",
  points: [],
  my_completed: [],
  partner_completed: [],
  days: [],
  locked: [],
  ...over,
});

describe("weekdayOf", () => {
  it("reads the local weekday, not UTC's", () => {
    // 2026-09-26 is a Saturday; parsed with year/month/day (not `new Date(iso)`) so it can never
    // shift a day depending on the reader's timezone offset.
    expect(weekdayOf("2026-09-26")).toBe(6);
    expect(weekdayOf("2026-09-20")).toBe(0); // Sunday
  });
});

describe("isForDay", () => {
  it("is true every day when weekdays is empty", () => {
    const p = point({ weekdays: [] });
    for (let d = 0; d < 7; d++) expect(isForDay(p, d)).toBe(true);
  });

  it("is true only on the chosen days", () => {
    const p = point({ weekdays: [1, 4] }); // Mon, Thu
    expect(isForDay(p, 1)).toBe(true);
    expect(isForDay(p, 4)).toBe(true);
    expect(isForDay(p, 0)).toBe(false);
    expect(isForDay(p, 2)).toBe(false);
  });
});

describe("todaysPoints", () => {
  it("is empty when the week has no `today` (a week from history)", () => {
    const w = week({ points: [point()], today: undefined });
    expect(todaysPoints(w)).toEqual([]);
  });

  it("includes every-day points on today's weekday", () => {
    const everyDay = point({ id: "a", weekdays: [] });
    const w = week({ points: [everyDay], today: "2026-09-26" }); // Saturday
    expect(todaysPoints(w)).toEqual([everyDay]);
  });

  it("includes only points scheduled for today's weekday", () => {
    const saturday = point({ id: "a", weekdays: [6] });
    const monday = point({ id: "b", weekdays: [1] });
    const w = week({ points: [saturday, monday], today: "2026-09-26" }); // Saturday
    expect(todaysPoints(w)).toEqual([saturday]);
  });
});

describe("startIndex", () => {
  it("opens at 0 when nothing is scheduled today", () => {
    const w = week({ points: [point({ weekdays: [1] })], today: "2026-09-26" }); // Saturday
    expect(startIndex(w, null)).toBe(0);
  });

  it("opens at the given `at` when it is a valid index into today's points", () => {
    const a = point({ id: "a", weekdays: [] });
    const b = point({ id: "b", weekdays: [] });
    const w = week({ points: [a, b], today: "2026-09-26" });
    expect(startIndex(w, "1")).toBe(1);
  });

  it("falls back to the first of today's points not yet prayed", () => {
    const a = point({ id: "a", weekdays: [] });
    const b = point({ id: "b", weekdays: [] });
    const w = week({ points: [a, b], my_completed: ["a"], today: "2026-09-26" });
    expect(startIndex(w, null)).toBe(1);
  });

  it("ignores an out-of-range or non-numeric `at`", () => {
    const a = point({ id: "a", weekdays: [] });
    const w = week({ points: [a], today: "2026-09-26" });
    expect(startIndex(w, "5")).toBe(0);
    expect(startIndex(w, "nope")).toBe(0);
  });

  it("is 0 for a week from history, since there is no `today` to index into", () => {
    const w = week({ points: [point()], today: undefined });
    expect(startIndex(w, "0")).toBe(0);
  });
});

describe("daysLabel", () => {
  it('reads "Every day" when nothing is chosen', () => {
    expect(daysLabel([])).toBe("Every day");
  });

  it("lists chosen days short and in order", () => {
    expect(daysLabel([4, 1])).toBe("Mon · Thu");
  });
});
