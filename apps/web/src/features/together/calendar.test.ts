import { describe, expect, it } from "vitest";

import type { Event, Milestone } from "@/lib/api/types";
import { nextAhead } from "./calendar";

const ev = (o: Partial<Event>): Event => ({
  id: "e",
  title: "Dinner at Nok",
  date: "2026-11-16",
  reminders: [],
  checklist: [],
  done: false,
  didnt_happen: false,
  kind: "together",
  created_by: null,
  ...o,
});
const ms = (o: Partial<Milestone>): Milestone => ({
  id: "m",
  title: "Her birthday",
  date: "1995-12-02",
  reminder: true,
  ...o,
});

describe("nextAhead", () => {
  it("is null when nothing is ahead", () => {
    expect(nextAhead([ev({ date: "2026-11-01" })], [ms({ reminder: false })], "2026-11-15")).toBeNull();
  });

  it("picks the earliest event on or after the start, skipping finished ones", () => {
    const list = [
      ev({ id: "a", date: "2026-11-20" }),
      ev({ id: "b", date: "2026-11-16" }),
      ev({ id: "c", date: "2026-11-15", done: true }),
      ev({ id: "d", date: "2026-11-15", didnt_happen: true }),
    ];
    const next = nextAhead(list, [], "2026-11-15");
    expect(next?.title).toBe("Dinner at Nok");
    expect(next?.date).toBe("2026-11-16");
  });

  it("prefers a reminded date that comes round sooner than any event", () => {
    const next = nextAhead([ev({ date: "2026-12-20" })], [ms({})], "2026-11-15");
    expect(next).toMatchObject({ title: "Her birthday", date: "2026-12-02" });
  });

  it("ignores a kept date with no reminder", () => {
    expect(nextAhead([], [ms({ reminder: false })], "2026-11-15")).toBeNull();
  });
});
