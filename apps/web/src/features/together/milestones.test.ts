import { describe, expect, it } from "vitest";

import type { Milestone } from "@/lib/api/types";
import {
  celebrationCopy,
  countdown,
  daysAway,
  daysAwayLabel,
  nextOccurrence,
  occursOn,
  todaysMilestones,
  upcomingMilestones,
  yearsBy,
} from "./milestones";

// Must agree with the server's notifications.OccursOn / EventReminderAt, or the calendar and the
// notification disagree on the day.
describe("nextOccurrence", () => {
  it("leaves a date that has not happened yet alone", () => {
    expect(nextOccurrence("2026-12-25", "2026-09-24")).toBe("2026-12-25");
  });

  it("brings a past date round to this year", () => {
    expect(nextOccurrence("2019-09-30", "2026-09-24")).toBe("2026-09-30");
  });

  it("and to next year once this year's has gone", () => {
    expect(nextOccurrence("2019-03-14", "2026-09-24")).toBe("2027-03-14");
  });

  it("counts today as still to come", () => {
    expect(nextOccurrence("2019-09-24", "2026-09-24")).toBe("2026-09-24");
  });

  it("moves the twenty-ninth of February to the twenty-eighth in a common year", () => {
    // 2027 has no 29th; "2027-02-29" would roll over to March 1 if left to the browser.
    expect(nextOccurrence("1996-02-29", "2026-09-24")).toBe("2027-02-28");
  });

  it("and leaves it alone in a leap year", () => {
    expect(nextOccurrence("1996-02-29", "2028-01-01")).toBe("2028-02-29");
  });
});

describe("occursOn", () => {
  it("matches the same day of any year", () => {
    expect(occursOn("2019-09-30", "2026-09-30")).toBe(true);
    expect(occursOn("2019-09-30", "2026-09-29")).toBe(false);
  });

  it("puts a leap day on the twenty-eighth when there is no twenty-ninth", () => {
    expect(occursOn("1996-02-29", "2027-02-28")).toBe(true);
    expect(occursOn("1996-02-29", "2027-03-01")).toBe(false);
  });

  it("and on the twenty-ninth when there is one", () => {
    expect(occursOn("1996-02-29", "2028-02-29")).toBe(true);
    expect(occursOn("1996-02-29", "2028-02-28")).toBe(false);
  });

  it("knows 2100 is not a leap year", () => {
    expect(occursOn("1996-02-29", "2100-02-28")).toBe(true);
  });
});

describe("yearsBy", () => {
  it("counts the years", () => {
    expect(yearsBy("2019-09-30", "2026-09-30")).toBe("7 years");
  });

  it("says one rather than 1", () => {
    expect(yearsBy("2025-09-30", "2026-09-30")).toBe("One year");
  });

  it("says nothing at all the first time round", () => {
    expect(yearsBy("2026-09-30", "2026-09-30")).toBe("");
  });

  it("says nothing for a birthday with no year on file, even years after its placeholder", () => {
    // 2000 is the placeholder year for a yearless birthday; without the flag this would say "26 years".
    expect(yearsBy("2000-09-30", "2026-09-30", false)).toBe("");
  });
});

describe("countdown", () => {
  const on = (iso: string) => new Date(`${iso}T09:00:00`);

  it("can say today", () => {
    expect(countdown("2026-09-24", on("2026-09-24"))).toBe("Today");
  });

  it("and tomorrow, rather than in 1 days", () => {
    expect(countdown("2026-09-25", on("2026-09-24"))).toBe("Tomorrow");
  });

  it("and counts the rest", () => {
    expect(countdown("2026-10-01", on("2026-09-24"))).toBe("in 7 days");
  });
});

describe("daysAway", () => {
  it("is 0 for today", () => {
    expect(daysAway("2019-09-24", "2026-09-24")).toBe(0);
  });

  it("counts up to the next occurrence", () => {
    expect(daysAway("2019-10-01", "2026-09-24")).toBe(7);
  });

  it("rolls a leap day to Feb 28 in a non-leap year, same as nextOccurrence", () => {
    expect(daysAway("1996-02-29", "2027-02-27")).toBe(1);
  });
});

describe("daysAwayLabel", () => {
  it("says tomorrow rather than in 1 days", () => {
    expect(daysAwayLabel(1)).toBe("tomorrow");
  });

  it("and counts the rest", () => {
    expect(daysAwayLabel(5)).toBe("in 5 days");
  });
});

const dateAt = (date: string, extra: Partial<Milestone> = {}): Milestone => ({
  id: date + JSON.stringify(extra),
  title: "Something",
  date,
  ...extra,
});

describe("todaysMilestones", () => {
  it("keeps only the ones landing today, leap day included", () => {
    const today = "2027-02-28";
    const dates = [
      dateAt("2019-02-28", { title: "On the day" }),
      dateAt("1996-02-29", { title: "Leap day, rolled" }),
      dateAt("2019-03-01", { title: "Not today" }),
    ];
    expect(todaysMilestones(dates, today).map((d) => d.title)).toEqual([
      "On the day",
      "Leap day, rolled",
    ]);
  });
});

describe("upcomingMilestones", () => {
  const today = "2026-09-24";

  it("excludes today and anything past the window, sorted soonest first", () => {
    const dates = [
      dateAt("2019-09-24", { title: "Today" }),
      dateAt("2019-09-30", { title: "In 6 days" }),
      dateAt("2019-09-26", { title: "In 2 days" }),
      dateAt("2019-10-05", { title: "Too far" }),
    ];
    expect(upcomingMilestones(dates, today).map((d) => d.title)).toEqual([
      "In 2 days",
      "In 6 days",
    ]);
  });
});

describe("celebrationCopy", () => {
  const today = "2026-09-24";

  it("greets a partner's birthday by name and asks you to tell them", () => {
    const m = dateAt("1998-09-24", { title: "Adeola's birthday", source: "birthday", about: "u2" });
    expect(celebrationCopy(m, today, "u1", "Adeola")).toEqual({
      title: "Happy birthday, Adeola",
      sub: "Tell them today.",
    });
  });

  it("greets your own birthday and credits your partner", () => {
    const m = dateAt("1998-09-24", { title: "My birthday", source: "birthday", about: "u1" });
    expect(celebrationCopy(m, today, "u1", "Adeola")).toEqual({
      title: "Happy birthday",
      sub: "Adeola has been waiting for today.",
    });
  });

  it("counts the years on an anniversary", () => {
    const m = dateAt("2019-09-24", { title: "Anniversary", source: "anniversary" });
    expect(celebrationCopy(m, today, "u1", "Adeola")).toEqual({
      title: "Happy anniversary",
      sub: "7 years together",
    });
  });

  it("falls back to the title for anything else", () => {
    const m = dateAt("2019-09-24", { title: "First trip abroad" });
    expect(celebrationCopy(m, today, "u1", "Adeola")).toEqual({
      title: "First trip abroad",
      sub: "Today",
    });
  });
});
