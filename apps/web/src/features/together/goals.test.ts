import { describe, expect, it } from "vitest";

import type { Goal } from "@/lib/api/types";
import { goalNothingYet } from "./goals";

const goal = (o: Partial<Goal>): Goal => ({ target: 1, unit: "count", ...o }) as Goal;

describe("goalNothingYet", () => {
  it("names a money target and asks for the first amount", () => {
    expect(goalNothingYet(goal({ unit: "naira", target: 1200000 }))).toBe(
      "Target ₦1,200,000 · add the first amount",
    );
  });

  it("names a count target with its label", () => {
    expect(goalNothingYet(goal({ target: 30, unit_label: "chapters" }))).toBe(
      "Target 30 chapters · add the first one",
    );
  });

  it("copes with a count goal that has no label", () => {
    expect(goalNothingYet(goal({ target: 30 }))).toBe("Target 30 · add the first one");
  });
});
