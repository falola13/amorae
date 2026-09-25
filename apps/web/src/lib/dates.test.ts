import { describe, expect, it } from "vitest";

import { addMonths, iso, monthGrid, parse, partOfDay, sameMonth, time12 } from "./dates";

describe("partOfDay", () => {
  // The card used to say "Tonight" over everything happening today, so an
  // 8:30 breakfast read "TONIGHT · 8:30 am".
  it("calls the morning the morning", () => {
    expect(partOfDay("08:30")).toBe("This morning");
    expect(partOfDay("00:05")).toBe("This morning");
    expect(partOfDay("11:59")).toBe("This morning");
  });

  it("the afternoon the afternoon", () => {
    expect(partOfDay("12:00")).toBe("This afternoon");
    expect(partOfDay("16:59")).toBe("This afternoon");
  });

  it("and only the evening tonight", () => {
    expect(partOfDay("17:00")).toBe("Tonight");
    expect(partOfDay("23:30")).toBe("Tonight");
  });

  it("says Today when there is no time to be more precise about", () => {
    expect(partOfDay()).toBe("Today");
    expect(partOfDay("")).toBe("Today");
  });

  it("and rather than guessing at something it cannot read", () => {
    expect(partOfDay("half past eight")).toBe("Today");
  });
});

describe("time12", () => {
  it("reads noon and midnight the way people say them", () => {
    expect(time12("00:00")).toBe("12:00 am");
    expect(time12("12:00")).toBe("12:00 pm");
  });

  it("keeps the minutes padded", () => {
    expect(time12("09:05")).toBe("9:05 am");
    expect(time12("19:30")).toBe("7:30 pm");
  });

  it("gives nothing back for no time, so a caller can skip the clause", () => {
    expect(time12()).toBe("");
    expect(time12("")).toBe("");
  });
});

describe("monthGrid", () => {
  it("covers the whole month in whole weeks, starting Sunday", () => {
    const grid = monthGrid(new Date(2026, 8, 15)); // September 2026
    expect(grid.length % 7).toBe(0);
    expect(parse(grid[0]).getDay()).toBe(0);
    expect(grid).toContain("2026-09-01");
    expect(grid).toContain("2026-09-30");
    // and it spills into the neighbouring months, which is what makes it a grid
    expect(grid[0] < "2026-09-01").toBe(true);
    expect(grid[grid.length - 1] > "2026-09-30").toBe(true);
  });

  it("handles a month that starts on a Sunday without an empty leading week", () => {
    // 1 Feb 2026 is a Sunday: the grid must not open with seven days of January.
    const grid = monthGrid(new Date(2026, 1, 10));
    expect(grid[0]).toBe("2026-02-01");
  });

  it("crosses a year boundary", () => {
    const grid = monthGrid(new Date(2026, 11, 1)); // December
    expect(grid).toContain("2026-12-31");
    expect(grid.some((d) => d.startsWith("2027-01"))).toBe(true);
  });
});

describe("addMonths", () => {
  it("does not roll over from a long month to a short one", () => {
    // Naive date arithmetic turns 31 Jan + 1 month into 3 March.
    expect(iso(addMonths(new Date(2026, 0, 31), 1))).toBe("2026-02-01");
  });

  it("steps backwards across a year", () => {
    expect(iso(addMonths(new Date(2026, 0, 15), -1))).toBe("2025-12-01");
  });
});

describe("sameMonth", () => {
  it("compares month, not day", () => {
    expect(sameMonth("2026-09-01", "2026-09-30")).toBe(true);
    expect(sameMonth("2026-09-30", "2026-10-01")).toBe(false);
    expect(sameMonth("2025-09-15", "2026-09-15")).toBe(false);
  });
});
