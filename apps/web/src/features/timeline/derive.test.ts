import { describe, expect, it } from "vitest";

import type { TimelineItem } from "@/lib/api/types";
import { groupByMonth } from "./derive";

function item(id: string, date: string): TimelineItem {
  return { id, type: "journal", at: `${date}T12:00:00Z`, date, title: id, sub: "", path: "/" };
}

describe("groupByMonth", () => {
  it("returns nothing for an empty list", () => {
    expect(groupByMonth([])).toEqual([]);
  });

  it("puts same-month items under one header", () => {
    const groups = groupByMonth([item("a", "2026-09-27"), item("b", "2026-09-01")]);
    expect(groups).toHaveLength(1);
    expect(groups[0].month).toBe("September 2026");
    expect(groups[0].items.map((i) => i.id)).toEqual(["a", "b"]);
  });

  it("starts a new group when the month changes", () => {
    const groups = groupByMonth([item("a", "2026-09-27"), item("b", "2026-08-30")]);
    expect(groups).toHaveLength(2);
    expect(groups.map((g) => g.month)).toEqual(["September 2026", "August 2026"]);
  });

  it("starts a fresh group for a month seen again after another one, rather than merging back into it", () => {
    const groups = groupByMonth([
      item("a", "2026-09-27"),
      item("b", "2026-08-30"),
      item("c", "2026-09-02"),
    ]);
    expect(groups).toHaveLength(3);
    expect(groups.map((g) => g.month)).toEqual(["September 2026", "August 2026", "September 2026"]);
  });
});
