import { describe, expect, it } from "vitest";

import type { AnniversaryYear, Appreciation, Memory } from "@/lib/api/types";
import {
  daysTogether,
  notesOfTheYear,
  photosOfTheYear,
  whenLabel,
  yearTiles,
  yearsWord,
} from "./derive";

const year: AnniversaryYear = {
  from: "2025-10-25",
  to: "2026-10-25",
  prayed: 312,
  answered: 0,
  prayer_weeks: 40,
  memories: 1,
  events: 0,
  challenges: 2,
  notes: 0,
  journal: 0,
};

describe("yearsWord", () => {
  it("spells the small ones and counts the rest", () => {
    expect(yearsWord(1)).toBe("One year");
    expect(yearsWord(3)).toBe("Three years");
    expect(yearsWord(12)).toBe("12 years");
  });
});

describe("whenLabel", () => {
  it("says when", () => {
    expect(whenLabel(0)).toBe("today");
    expect(whenLabel(1)).toBe("tomorrow");
    expect(whenLabel(6)).toBe("in 6 days");
  });
});

describe("daysTogether", () => {
  it("counts the days since they started", () => {
    expect(daysTogether("2023-10-25", "2026-10-25")).toBe(1096);
    expect(daysTogether("2026-10-25", "2026-10-25")).toBe(0);
  });
});

describe("yearTiles", () => {
  it("leaves out what is zero and gets the singular right", () => {
    expect(yearTiles(year)).toEqual([
      { n: 312, label: "prayers prayed" },
      { n: 40, label: "weeks of prayer" },
      { n: 1, label: "memory kept" },
      { n: 2, label: "challenges finished" },
    ]);
  });
});

describe("notesOfTheYear", () => {
  const note = (date: string): Appreciation => ({ id: date, from_id: "a", date, text: date });
  it("keeps to the year and spreads across it", () => {
    const notes = [
      "2025-10-24",
      "2025-11-01",
      "2026-01-01",
      "2026-03-01",
      "2026-05-01",
      "2026-07-01",
      "2026-10-25",
    ].map(note);
    const got = notesOfTheYear(notes, year, 3).map((n) => n.date);
    expect(got).toEqual(["2025-11-01", "2026-03-01", "2026-07-01"]);
  });
});

describe("photosOfTheYear", () => {
  it("only memories with a photo", () => {
    const m = (date: string, photo: boolean): Memory => ({
      id: date,
      title: date,
      date,
      has_photo: photo,
      photo_url: photo ? "https://example.com/p.jpg" : undefined,
    });
    expect(
      photosOfTheYear([m("2026-01-01", true), m("2026-02-01", false)], year).map((x) => x.id),
    ).toEqual(["2026-01-01"]);
  });
});
