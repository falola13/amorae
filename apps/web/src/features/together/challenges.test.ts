import { describe, expect, it } from "vitest";

import type { Challenge, ChallengeDay, ChallengeTemplate } from "@/lib/api/types";
import {
  currentSeason,
  customProblem,
  dayDateLabel,
  easterSunday,
  groupTemplates,
  homeChallengeDay,
  isCatchUp,
  opensLabel,
  pickSuggestions,
  resizePrompts,
  shownDay,
} from "./challenges";

const day = (n: number, over: Partial<ChallengeDay> = {}): ChallengeDay => ({
  n,
  text: `Day ${n}`,
  date: "2026-10-01",
  open: true,
  done: false,
  ...over,
});
const challenge = (over: Partial<Challenge> = {}): Challenge => ({
  id: "c1",
  template: "t",
  title: "Seven",
  status: "active",
  started_on: "2026-09-28",
  ended_at: null,
  today_n: 2,
  created_by: null,
  days: [day(1), day(2), day(3, { open: false })],
  ...over,
});
const tpl = (key: string, over: Partial<ChallengeTemplate> = {}): ChallengeTemplate => ({
  key,
  title: key,
  blurb: "",
  days: 7,
  category: "connection",
  season: "",
  ...over,
});

describe("easterSunday", () => {
  it("finds known dates", () => {
    expect(easterSunday(2026).getMonth()).toBe(3);
    expect(easterSunday(2026).getDate()).toBe(5);
    expect(easterSunday(2025).getDate()).toBe(20);
    expect(easterSunday(2024).getMonth()).toBe(2);
    expect(easterSunday(2024).getDate()).toBe(31);
  });
});

describe("currentSeason", () => {
  it("is Advent on Dec 1 to 24", () => {
    expect(currentSeason(new Date(2026, 11, 1))).toBe("advent");
    expect(currentSeason(new Date(2026, 11, 24))).toBe("advent");
    expect(currentSeason(new Date(2026, 11, 25))).toBe("");
  });
  it("is Lent from Ash Wednesday to Holy Saturday 2026", () => {
    expect(currentSeason(new Date(2026, 1, 18))).toBe("lent");
    expect(currentSeason(new Date(2026, 3, 4))).toBe("lent");
    expect(currentSeason(new Date(2026, 1, 17))).toBe("");
    expect(currentSeason(new Date(2026, 3, 5))).toBe("");
  });
  it("is neither in an ordinary week", () => {
    expect(currentSeason(new Date(2026, 8, 30))).toBe("");
  });
});

describe("opensLabel", () => {
  it("says tomorrow, then the weekday, then the date", () => {
    expect(opensLabel("2026-10-01", "2026-09-30")).toBe("Opens tomorrow");
    expect(opensLabel("2026-10-03", "2026-09-30")).toBe("Opens Saturday");
    expect(opensLabel("2026-10-10", "2026-09-30")).toBe("Opens Saturday 10 October");
  });
  it("formats a day's date", () => {
    expect(dayDateLabel("2026-10-12")).toBe("Mon 12 Oct");
  });
});

describe("isCatchUp", () => {
  it("is an earlier open day with no mark", () => {
    expect(isCatchUp(day(1), 2)).toBe(true);
  });
  it("is not today, a marked day, or a locked one", () => {
    expect(isCatchUp(day(2), 2)).toBe(false);
    expect(isCatchUp(day(1, { done: true }), 2)).toBe(false);
    expect(isCatchUp(day(1, { skipped: true }), 2)).toBe(false);
    expect(isCatchUp(day(3, { open: false }), 4)).toBe(false);
  });
});

describe("shownDay", () => {
  it("never runs past the last day", () => {
    expect(shownDay(challenge({ today_n: 9 }))).toBe(3);
    expect(shownDay(challenge({ today_n: 2 }))).toBe(2);
  });
});

describe("homeChallengeDay", () => {
  it("offers today's day when you have not marked it", () => {
    expect(homeChallengeDay(challenge())?.n).toBe(2);
  });
  it("hides once you marked or skipped it", () => {
    const done = challenge({ days: [day(1), day(2, { done: true })] });
    const skipped = challenge({ days: [day(1), day(2, { skipped: true })] });
    expect(homeChallengeDay(done)).toBeNull();
    expect(homeChallengeDay(skipped)).toBeNull();
  });
  it("hides when today is locked, the challenge is over, or there is none", () => {
    expect(homeChallengeDay(challenge({ days: [day(1), day(2, { open: false })] }))).toBeNull();
    expect(homeChallengeDay(challenge({ status: "finished" }))).toBeNull();
    expect(homeChallengeDay(undefined)).toBeNull();
  });
});

describe("pickSuggestions", () => {
  const all = [
    tpl("a"),
    tpl("b"),
    tpl("c"),
    tpl("d"),
    tpl("adv", { category: "season", season: "advent" }),
    tpl("len", { category: "season", season: "lent" }),
  ];
  const ordinary = new Date(2026, 8, 30);

  it("prefers ones not done before", () => {
    expect(pickSuggestions(all, ["a", "b"], ordinary).map((t) => t.key)).toEqual(["c", "d", "a"]);
  });
  it("never suggests another season's challenge", () => {
    const keys = pickSuggestions(all, [], ordinary, 10).map((t) => t.key);
    expect(keys).not.toContain("adv");
    expect(keys).not.toContain("len");
  });
  it("puts the season's own first", () => {
    expect(pickSuggestions(all, [], new Date(2026, 11, 3)).map((t) => t.key)).toEqual([
      "adv",
      "a",
      "b",
    ]);
    expect(pickSuggestions(all, [], new Date(2026, 2, 10)).map((t) => t.key)[0]).toBe("len");
  });
  it("falls back to done ones when nothing else is left", () => {
    expect(pickSuggestions([tpl("a"), tpl("b")], ["a", "b"], ordinary)).toHaveLength(2);
  });
});

describe("groupTemplates", () => {
  const all = [
    tpl("a"),
    tpl("f", { category: "faith" }),
    tpl("adv", { category: "season", season: "advent" }),
    tpl("len", { category: "season", season: "lent" }),
  ];
  it("puts the season group first while it is on, this season's own first inside it", () => {
    const g = groupTemplates(all, "lent");
    expect(g.map((x) => x.category)).toEqual(["season", "connection", "faith"]);
    expect(g[0].templates.map((t) => t.key)).toEqual(["len", "adv"]);
  });
  it("keeps it last, but present, out of season", () => {
    expect(groupTemplates(all, "").map((x) => x.category)).toEqual([
      "connection",
      "faith",
      "season",
    ]);
  });
  it("drops empty groups", () => {
    expect(groupTemplates([tpl("a")], "").map((x) => x.category)).toEqual(["connection"]);
  });
});

describe("custom challenge helpers", () => {
  it("resizes keeping what is written", () => {
    expect(resizePrompts(["a", "b"], 4)).toEqual(["a", "b", "", ""]);
    expect(resizePrompts(["a", "b", "c", "d"], 3)).toEqual(["a", "b", "c"]);
  });
  it("keeps the count within 3 to 40", () => {
    expect(resizePrompts([], 1)).toHaveLength(3);
    expect(resizePrompts([], 99)).toHaveLength(40);
    expect(resizePrompts([], Number.NaN)).toHaveLength(3);
  });
  it("names the first problem", () => {
    expect(customProblem(" ", ["a", "b", "c"])).toBe("Give it a name.");
    expect(customProblem("T", ["a", "", "c"])).toBe("Each day needs a line.");
    expect(customProblem("T", ["a", "b", "c"])).toBeNull();
  });
});
