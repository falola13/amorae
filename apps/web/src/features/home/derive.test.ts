import { describe, expect, it } from "vitest";

import { ApiError } from "@/lib/api/errors";
import type { Event } from "@/lib/api/types";
import { isAlone, todayEvent, upcomingEvents } from "./derive";

const apiError = (code: string) => new ApiError(409, code, "");

describe("isAlone", () => {
  it("is true with no partner, even though nothing failed", () => {
    expect(isAlone(false, undefined)).toBe(true);
  });

  it("is true when the API says so", () => {
    expect(isAlone(false, apiError("waiting_for_partner"))).toBe(true);
    expect(isAlone(true, apiError("waiting_for_partner"))).toBe(true);
  });

  it("is false once a partner is there and nothing is wrong", () => {
    expect(isAlone(true, undefined)).toBe(false);
  });

  it("does not mistake an unrelated failure for being alone", () => {
    expect(isAlone(true, apiError("internal_error"))).toBe(false);
    expect(isAlone(true, new Error("network"))).toBe(false);
  });
});

describe("upcomingEvents", () => {
  const ev = (o: Partial<Event>): Event => ({
    id: "1",
    title: "Dinner",
    date: "2026-09-30",
    reminders: [],
    checklist: [],
    done: false,
    didnt_happen: false,
    kind: "together",
    created_by: null,
    ...o,
  });
  const now = new Date(2026, 8, 30, 19, 0);

  it("keeps ongoing and upcoming events, earliest first, and drops over ones", () => {
    const list = [
      ev({ id: "late", start_time: "21:00" }),
      ev({ id: "now", start_time: "18:00", end_time: "21:00" }),
      ev({ id: "over", start_time: "10:00", end_time: "11:00" }),
      ev({ id: "done", start_time: "20:00", done: true }),
      ev({ id: "no", start_time: "20:00", didnt_happen: true }),
      ev({ id: "tomorrow", date: "2026-10-01" }),
    ];
    expect(upcomingEvents(list, now).map((e) => e.id)).toEqual(["now", "late", "tomorrow"]);
  });

  it("keeps a no-time event on its day and picks it as today's", () => {
    const up = upcomingEvents([ev({ id: "all-day" })], now);
    expect(todayEvent(up, "2026-09-30")?.id).toBe("all-day");
  });
});
