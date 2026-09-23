import { longDate, naira } from "@/lib/dates";
import type { Goal } from "@/lib/api/types";

export const sumOf = (g: Goal) => g.progress.reduce((s, p) => s + p.amount, 0);
export const goalSub = (g: Goal) =>
  g.unit === "naira"
    ? `${naira(sumOf(g))} of ${naira(g.target)} · by ${longDate(g.end_date)}`
    : `${sumOf(g)} of ${g.target} ${g.unit_label ?? ""} · by ${longDate(g.end_date)}`;
