import { describe, expect, it } from "vitest";

import { countdown, nextOccurrence, occursOn, yearsBy } from "./milestones";

// These four decide which day a kept date lands on, and the server decides
// the same thing again when it sends the reminder
// (notifications.OccursOn / EventReminderAt). If the two ever disagree the
// calendar counts down to one day and the notification arrives on another,
// which is the sort of thing nobody notices until February.
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
    // 2027 has no 29th. Building "2027-02-29" and letting the browser sort
    // it out lands on 1 March, a day late and in the wrong month.
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
});

describe("countdown", () => {
  const on = (iso: string) => new Date(`${iso}T09:00:00`);

  it("can say today", () => {
    // It used to read "in 0 days" on the one day that matters.
    expect(countdown("2026-09-24", on("2026-09-24"))).toBe("Today");
  });

  it("and tomorrow, rather than in 1 days", () => {
    expect(countdown("2026-09-25", on("2026-09-24"))).toBe("Tomorrow");
  });

  it("and counts the rest", () => {
    expect(countdown("2026-10-01", on("2026-09-24"))).toBe("in 7 days");
  });
});
