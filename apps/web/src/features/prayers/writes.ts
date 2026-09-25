import type { PrayerPoint, PrayerWeek } from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { defineWrite } from "@/lib/query/mutations";
import { prayersApi } from "./api";

export type CompletionVars = { pointId: string; done: boolean };

// Every write the prayers feature makes, declared once (see lib/query/mutations.ts).
export const prayerWrites = {
  // Toggles share a scope so they're sent one at a time, in order:
  // "prayed, then undo" can never land reversed.
  completion: defineWrite({
    mutationKey: ["prayers", "completion"],
    mutationFn: ({ pointId, done }: CompletionVars) =>
      done ? prayersApi.complete(pointId) : prayersApi.uncomplete(pointId),
    invalidates: [keys.week],
    scope: "prayers.completion",
    idempotent: "one row per (point, person), so marking or unmarking twice settles the same way.",
  }),
  savePoints: defineWrite({
    mutationKey: ["prayers", "savePoints"],
    mutationFn: (points: PrayerPoint[]) => prayersApi.savePoints(points),
    invalidates: [keys.week],
    idempotent: "sends the whole week, so a replay rewrites the same list rather than appending.",
  }),
  publish: defineWrite<void, PrayerWeek>({
    mutationKey: ["prayers", "publish"],
    mutationFn: () => prayersApi.publish(),
    invalidates: [keys.week, keys.history],
    idempotent: "publishing a week that is already published changes nothing.",
  }),
  // Note is sent with the mark, not after — two round trips would leave a moment where it's answered
  // but the reason is missing.
  answer: defineWrite({
    mutationKey: ["prayers", "answer"],
    mutationFn: ({ pointId, note }: { pointId: string; note: string }) =>
      prayersApi.setAnswered(pointId, note),
    invalidates: [keys.week, keys.history, keys.weeksById, keys.answered],
    scope: "prayers.answer",
    idempotent: "sets one point to answered with a given note, so a replay lands the same state.",
    optimistic: (qc, { pointId, note }) => {
      const at = new Date().toISOString();
      qc.setQueryData<PrayerWeek>(keys.week, (w) =>
        w
          ? {
              ...w,
              points: w.points.map((p) =>
                p.id === pointId ? { ...p, answered_at: at, answer_note: note } : p,
              ),
            }
          : w,
      );
    },
  }),
  unanswer: defineWrite({
    mutationKey: ["prayers", "unanswer"],
    mutationFn: ({ pointId }: { pointId: string }) => prayersApi.unsetAnswered(pointId),
    invalidates: [keys.week, keys.history, keys.weeksById, keys.answered],
    scope: "prayers.answer",
    idempotent: "clears the mark, which is already clear the second time.",
    optimistic: (qc, { pointId }) => {
      qc.setQueryData<PrayerWeek>(keys.week, (w) =>
        w
          ? {
              ...w,
              points: w.points.map((p) =>
                p.id === pointId ? { ...p, answered_at: undefined, answer_note: undefined } : p,
              ),
            }
          : w,
      );
    },
  }),
  reflection: defineWrite({
    mutationKey: ["prayers", "reflection"],
    mutationFn: ({ weekId, text }: { weekId: string; text: string }) =>
      prayersApi.reflection(weekId, text),
    invalidates: [keys.history, keys.week, keys.weeksById],
    idempotent: "one reflection per person per week, replaced rather than added to.",
  }),
};
