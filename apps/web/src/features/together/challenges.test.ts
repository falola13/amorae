import { describe, expect, it } from "vitest";

import type { Challenge, ChallengeDay, ChallengeTemplate } from "@/lib/api/types";
import {
  applyLine,
  atLimit,
  cheerOn,
  isMarkable,
  ownerLabel,
  planChanged,
  planFromText,
  planProblem,
  progressLabel,
  startConfirm,
  startProblem,
  startedOnToSend,
  currentSeason,
  challengesLine,
  customProblem,
  customPrompts,
  linesToPrompts,
  dayDateLabel,
  easterSunday,
  groupTemplates,
  homeChallengeDay,
  homeChallengeRows,
  isCatchUp,
  opensLabel,
  pickSuggestions,
  runningKeys,
  shownDay,
  todaysDay,
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
  kind: "together",
  starts_in: 0,
  can_edit: true,
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

describe("homeChallengeRows", () => {
  it("lists each running challenge whose today is open and unmarked", () => {
    const a = challenge({ id: "a" });
    const b = challenge({ id: "b", days: [day(1), day(2, { done: true })] });
    const c = challenge({ id: "c", status: "finished" });
    const d = challenge({ id: "d" });
    expect(homeChallengeRows([a, b, c, d]).map((r) => r.c.id)).toEqual(["a", "d"]);
    expect(homeChallengeRows([a, b, c, d])[0].day.n).toBe(2);
  });
  it("is empty with none, or when everything is marked", () => {
    expect(homeChallengeRows(undefined)).toEqual([]);
    expect(homeChallengeRows([])).toEqual([]);
    expect(homeChallengeRows([challenge({ days: [day(1), day(2, { skipped: true })] })])).toEqual(
      [],
    );
  });
  it("never shows more than three", () => {
    const list = ["a", "b", "c", "d"].map((id) => challenge({ id }));
    expect(homeChallengeRows(list)).toHaveLength(3);
  });
});

describe("running state", () => {
  it("collects the templates that are running", () => {
    const keys = runningKeys([
      challenge({ template: "x" }),
      challenge({ template: "y", status: "ended" }),
    ]);
    expect(keys.has("x")).toBe(true);
    expect(keys.has("y")).toBe(false);
    expect(runningKeys(undefined).size).toBe(0);
  });
  it("finds today's day, clamped to the last", () => {
    expect(todaysDay(challenge())?.n).toBe(2);
    expect(todaysDay(challenge({ today_n: 9 }))?.n).toBe(3);
  });
  it("words the hub line by how many run", () => {
    expect(challengesLine([])).toBe("Something short, together");
    expect(challengesLine([challenge()])).toBe("Seven, day 2 of 3");
    expect(challengesLine([challenge(), challenge({ id: "c2" })])).toBe("2 running");
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
  it("repeats one line for a habit — thirty days of the same thing", () => {
    const p = customPrompts("same", "  No soda, fruit only ", 30, "");
    expect(p).toHaveLength(30);
    expect(new Set(p)).toEqual(new Set(["No soda, fruit only"]));
  });
  it("takes a pasted list one line per day, ignoring blanks", () => {
    expect(linesToPrompts("Pray\r\n\n  Cook \nWalk\n")).toEqual(["Pray", "Cook", "Walk"]);
    expect(customPrompts("each", "ignored", 7, "a\nb\nc")).toEqual(["a", "b", "c"]);
  });
  it("allows up to a hundred days, not past it", () => {
    expect(customProblem("T", customPrompts("same", "x", 100, ""))).toBeNull();
    expect(customProblem("T", customPrompts("same", "x", 101, ""))).toBe(
      "Choose between 3 and 100 days.",
    );
    expect(customProblem("T", customPrompts("same", "x", Number.NaN, ""))).toBe(
      "Choose between 3 and 100 days.",
    );
  });
  it("names the first problem", () => {
    expect(customProblem(" ", ["a", "b", "c"])).toBe("Give it a name.");
    expect(customProblem("T", ["a", "", "c"])).toBe("Each day needs a line.");
    expect(customProblem("T", customPrompts("same", " ", 7, ""))).toBe("Each day needs a line.");
    expect(customProblem("T", ["a", "b", "x".repeat(201)])).toBe(
      "Keep each day under 200 characters.",
    );
    expect(customProblem("T", ["a", "b", "c"])).toBeNull();
  });
});

describe("marking rights", () => {
  const theirs = { kind: "mine" as const, can_edit: false };
  it("a partner's own challenge is read-only", () => {
    expect(isMarkable(theirs)).toBe(false);
    expect(isMarkable({ kind: "mine", can_edit: true })).toBe(true);
    expect(isMarkable({ kind: "together", can_edit: false })).toBe(true);
  });
  it("labels who it belongs to", () => {
    expect(ownerLabel(theirs, "Adeola")).toBe("Adeola’s");
    expect(ownerLabel({ kind: "mine", can_edit: true }, "Adeola")).toBe("Just you");
    expect(ownerLabel({ kind: "together", can_edit: true }, "Adeola")).toBeNull();
    expect(cheerOn("Adeola")).toBe("Adeola’s own — cheer them on.");
  });
  it("keeps a partner's own out of Home, the limit and the running keys", () => {
    const theirsC = challenge({ id: "p", kind: "mine", can_edit: false });
    expect(homeChallengeRows([theirsC])).toEqual([]);
    expect(runningKeys([theirsC]).size).toBe(0);
    expect(atLimit([theirsC, theirsC, theirsC])).toBe(false);
    expect(
      atLimit([challenge(), challenge({ id: "b" }), challenge({ id: "c", kind: "mine" })]),
    ).toBe(true);
  });
});

describe("a scheduled challenge", () => {
  const sched = challenge({
    starts_in: 3,
    today_n: 0,
    started_on: "2026-10-04",
    days: [day(1, { open: false }), day(2, { open: false }), day(3, { open: false })],
  });
  it("is not on Home and has no day today", () => {
    expect(homeChallengeDay(sched)).toBeNull();
    expect(homeChallengeRows([sched])).toEqual([]);
    expect(todaysDay(sched)).toBeUndefined();
  });
  it("says when it starts instead of a day number", () => {
    expect(progressLabel(sched)).toBe("Starts Sunday 4 October · in 3 days");
    expect(progressLabel(challenge())).toBe("Day 2 of 3");
    expect(challengesLine([sched])).toBe("Seven, starts sunday 4 october");
  });
});

describe("the plan", () => {
  it("reads one line per day, ignoring blanks", () => {
    expect(planFromText("a\n\n b \r\nc")).toEqual(["a", "b", "c"]);
  });
  it("rewrites every day, or only from a day on", () => {
    expect(applyLine(["a", "b", "c", "d"], " x ", 1)).toEqual(["x", "x", "x", "x"]);
    expect(applyLine(["a", "b", "c", "d"], "x", 3)).toEqual(["a", "b", "x", "x"]);
    expect(applyLine(["a", "b"], "  ", 1)).toEqual(["a", "b"]);
  });
  it("notices a change in length or text", () => {
    expect(planChanged(["a", "b", "c"], ["a", "b", "c"])).toBe(false);
    expect(planChanged(["a", "b", "c"], ["a", "b", "x"])).toBe(true);
    expect(planChanged(["a", "b", "c"], ["a", "b", "c", "d"])).toBe(true);
  });
  it("needs three to a hundred days", () => {
    expect(planProblem(["a", "b"])).toMatch(/between 3 and 100/);
    expect(planProblem(["a", "b", "c"])).toBeNull();
  });
});

describe("starting", () => {
  const now = new Date(2026, 9, 1);
  it("accepts today to sixty days out", () => {
    expect(startProblem("2026-10-01", now)).toBeNull();
    expect(startProblem("2026-11-30", now)).toBeNull();
    expect(startProblem("2026-12-01", now)).not.toBeNull();
    expect(startProblem("2026-09-30", now)).not.toBeNull();
    expect(startProblem("", now)).not.toBeNull();
  });
  it("sends a date only when it is after today", () => {
    expect(startedOnToSend("2026-10-01", now)).toBeUndefined();
    expect(startedOnToSend("2026-10-05", now)).toBe("2026-10-05");
  });
  it("confirms in the right words", () => {
    expect(startConfirm("together", undefined)).toBe("Day 1 opens today for both of you.");
    expect(startConfirm("mine", undefined)).toBe("Day 1 opens today — just for you.");
    expect(startConfirm("together", "2026-10-05")).toBe("Day 1 opens Monday 5 October.");
  });
});
