import { describe, expect, it } from "vitest";

import { partOfDay, time12 } from "./dates";

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
