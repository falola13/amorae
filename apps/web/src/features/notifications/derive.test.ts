import { describe, expect, it } from "vitest";

import type { NotificationItem } from "@/lib/api/types";
import { nudgeLine, nudgesToday } from "./derive";

const at = (h: number, m: number, day = 15) => new Date(2026, 10, day, h, m);
const item = (id: string, kind: NotificationItem["kind"], when: Date): NotificationItem => ({
  id,
  kind,
  title: "",
  body: "",
  path: "/",
  created_at: when.toISOString(),
  read: false,
});

describe("nudgesToday", () => {
  it("is null when there are none today", () => {
    expect(nudgesToday([], "2026-11-15")).toBeNull();
    expect(
      nudgesToday([item("a", "nudge", at(9, 0, 14)), item("b", "journal", at(9, 0))], "2026-11-15"),
    ).toBeNull();
  });

  it("counts today's nudges and finds the latest, whatever the order", () => {
    const got = nudgesToday(
      [
        item("a", "nudge", at(9, 5)),
        item("b", "nudge", at(15, 12)),
        item("c", "nudge", at(11, 0)),
        item("d", "appreciation", at(16, 0)),
      ],
      "2026-11-15",
    );
    expect(got?.count).toBe(3);
    expect(got?.last.getHours()).toBe(15);
  });
});

describe("nudgeLine", () => {
  it("says it once with the time", () => {
    expect(nudgeLine("Ada", 1, at(15, 12))).toBe("Ada thought of you · 3:12 pm");
  });

  it("counts several and gives the last", () => {
    expect(nudgeLine("Ada", 3, at(15, 12))).toBe("Ada thought of you 3 times today · last 3:12 pm");
  });

  it("reads midnight and morning right", () => {
    expect(nudgeLine("Ada", 1, at(0, 5))).toBe("Ada thought of you · 12:05 am");
  });
});
