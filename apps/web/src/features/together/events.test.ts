import { describe, expect, it } from "vitest";

import {
  endsLabel,
  eventPhase,
  eventReminderLabel,
  nowLabel,
  remindersSummary,
  type EventPhase,
} from "./events";

type E = Parameters<typeof eventPhase>[0];
const ev = (o: Partial<E> = {}): E => ({
  date: "2026-09-30",
  done: false,
  didnt_happen: false,
  ...o,
});
const at = (h: number, m = 0, day = 30, s = 0) => new Date(2026, 8, day, h, m, s);
const phase = (e: Partial<E>, now: Date): EventPhase => eventPhase(ev(e), now);

describe("eventPhase with no times", () => {
  it("is ongoing all of its day, upcoming before, over after", () => {
    expect(phase({}, at(0, 0, 29, 59))).toBe("upcoming");
    expect(phase({}, at(0, 0))).toBe("ongoing");
    expect(phase({}, at(23, 58))).toBe("ongoing");
    expect(phase({}, at(0, 0, 31))).toBe("over");
  });
});

describe("eventPhase with only a start", () => {
  const e = { start_time: "18:00" };
  it("is upcoming before the start", () => {
    expect(phase(e, at(17, 59))).toBe("upcoming");
  });
  it("is ongoing from the start (inclusive) for two hours", () => {
    expect(phase(e, at(18, 0))).toBe("ongoing");
    expect(phase(e, at(19, 59))).toBe("ongoing");
  });
  it("is over at exactly two hours", () => {
    expect(phase(e, at(20, 0))).toBe("over");
  });
  it("runs two hours across midnight", () => {
    const late = { start_time: "23:00" };
    expect(phase(late, at(0, 30, 31))).toBe("ongoing");
    expect(phase(late, at(1, 0, 31))).toBe("over");
  });
});

describe("eventPhase with a start and an end", () => {
  const e = { start_time: "18:00", end_time: "21:00" };
  it("is ongoing until the end, then over", () => {
    expect(phase(e, at(17, 59))).toBe("upcoming");
    expect(phase(e, at(18, 0))).toBe("ongoing");
    expect(phase(e, at(20, 59, 30, 59))).toBe("ongoing");
    expect(phase(e, at(21, 0))).toBe("over");
  });
  it("can be longer than two hours", () => {
    expect(phase({ start_time: "09:00", end_time: "17:00" }, at(15, 0))).toBe("ongoing");
  });
});

describe("eventPhase overnight", () => {
  const e = { start_time: "22:00", end_time: "02:00" };
  it("ends the next day when the end is not after the start", () => {
    expect(phase(e, at(21, 59))).toBe("upcoming");
    expect(phase(e, at(23, 30))).toBe("ongoing");
    expect(phase(e, at(1, 59, 31))).toBe("ongoing");
    expect(phase(e, at(2, 0, 31))).toBe("over");
  });
  it("treats an end equal to the start as the next day", () => {
    expect(phase({ start_time: "10:00", end_time: "10:00" }, at(15, 0))).toBe("ongoing");
  });
});

describe("eventPhase with an end but no start", () => {
  it("runs from midnight to the end time", () => {
    const e = { end_time: "12:00" };
    expect(phase(e, at(0, 0))).toBe("ongoing");
    expect(phase(e, at(11, 59))).toBe("ongoing");
    expect(phase(e, at(12, 0))).toBe("over");
  });
});

describe("eventPhase outcome", () => {
  it("is always over once done or marked as not happened", () => {
    expect(phase({ done: true }, at(8))).toBe("over");
    expect(phase({ didnt_happen: true, date: "2026-10-05" }, at(8))).toBe("over");
  });
  it("is upcoming for a future day", () => {
    expect(phase({ date: "2026-10-05", start_time: "09:00" }, at(8))).toBe("upcoming");
  });
  it("is over for a past day", () => {
    expect(phase({ date: "2026-09-28", start_time: "09:00" }, at(8))).toBe("over");
  });
});

describe("labels", () => {
  it("says when it ends", () => {
    expect(endsLabel({ end_time: "21:00" })).toBe("until 9:00 pm");
    expect(endsLabel({})).toBe("");
  });
  it("says now, with the end when there is one", () => {
    expect(nowLabel({ end_time: "21:00" })).toBe("Now · until 9:00 pm");
    expect(nowLabel({})).toBe("Now");
  });
  it("reads reminders in plain words", () => {
    expect(eventReminderLabel("1 hour before")).toBe("1 hour before");
    expect(eventReminderLabel("1 day before")).toBe("The day before");
    expect(eventReminderLabel("at 16:00")).toBe("At 4:00 pm");
    expect(eventReminderLabel("some old text")).toBe("Some old text");
  });
  it("sums up reminders", () => {
    expect(remindersSummary([])).toBe("");
    expect(remindersSummary(["at the time"])).toBe("At the time");
    expect(remindersSummary(["at the time", "at 09:00"])).toBe("2 reminders");
  });
});
