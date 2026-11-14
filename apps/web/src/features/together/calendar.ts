import type { Event, Milestone } from "@/lib/api/types";
import { routes } from "@/lib/routes";
import { nextOccurrence } from "./milestones";

export type Ahead = { title: string; date: string; href: string };

/** The nearest plan, or kept date with a reminder, on or after `from` — what to
 *  point at when the part of the calendar being looked at is empty. An event wins
 *  a tie, since it has a page of its own. */
export function nextAhead(events: Event[], milestones: Milestone[], from: string): Ahead | null {
  let best: Ahead | null = null;
  const consider = (a: Ahead) => {
    if (!best || a.date < best.date) best = a;
  };
  for (const e of events)
    if (!e.done && !e.didnt_happen && e.date >= from)
      consider({ title: e.title, date: e.date, href: routes.event(e.id) });
  for (const m of milestones)
    if (m.reminder)
      consider({
        title: m.title,
        date: nextOccurrence(m.date, from),
        href: routes.milestones,
      });
  return best;
}
