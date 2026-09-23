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
  }),
  savePoints: defineWrite({
    mutationKey: ["prayers", "savePoints"],
    mutationFn: (points: PrayerPoint[]) => prayersApi.savePoints(points),
    invalidates: [keys.week],
  }),
  publish: defineWrite<void, PrayerWeek>({
    mutationKey: ["prayers", "publish"],
    mutationFn: () => prayersApi.publish(),
    invalidates: [keys.week, keys.history],
  }),
  reflection: defineWrite({
    mutationKey: ["prayers", "reflection"],
    mutationFn: ({ weekId, text }: { weekId: string; text: string }) =>
      prayersApi.reflection(weekId, text),
    invalidates: [keys.history, keys.week, keys.weeksById],
  }),
};
