"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { PrayerWeek } from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { useWrite } from "@/lib/query/mutations";
import { usePausedVariables } from "@/lib/query/offline";
import { useCouple } from "@/features/couple/hooks";
import { prayersApi } from "./api";
import { prayerWrites, type CompletionVars } from "./writes";

// Disabled until a partner exists — otherwise "waiting for partner" would surface as a query error.
export function useWeek() {
  const couple = useCouple();
  return useQuery({
    queryKey: keys.week,
    queryFn: prayersApi.current,
    enabled: Boolean(couple.data?.partner),
  });
}
export const useHistory = () => useQuery({ queryKey: keys.history, queryFn: prayersApi.history });
export const useWeekById = (id: string) =>
  useQuery({ queryKey: keys.weekById(id), queryFn: () => prayersApi.week(id), enabled: !!id });

const withCompletion = (w: PrayerWeek, { pointId, done }: CompletionVars): PrayerWeek => ({
  ...w,
  my_completed: done
    ? Array.from(new Set([...w.my_completed, pointId]))
    : w.my_completed.filter((x) => x !== pointId),
});

/**
 * Optimistic: the check draws in at once. Offline, the call is paused, kept
 * across restarts, and sent on reconnect (lib/query/client.ts, persist.ts).
 */
export function useSetCompleted() {
  const qc = useQueryClient();
  const def = prayerWrites.completion;
  return useMutation({
    mutationKey: def.mutationKey,
    mutationFn: def.mutationFn,
    scope: { id: def.scope! },
    onMutate: async (vars) => {
      await qc.cancelQueries({ queryKey: keys.week });
      const prev = qc.getQueryData<PrayerWeek>(keys.week);
      if (prev) qc.setQueryData<PrayerWeek>(keys.week, withCompletion(prev, vars));
      return { prev };
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.prev) qc.setQueryData(keys.week, ctx.prev);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: keys.week }),
  });
}

/** Prayer ids whose "prayed" change is waiting for the connection. */
export function usePendingCompletions(): Set<string> {
  return new Set(
    usePausedVariables<CompletionVars>(prayerWrites.completion.mutationKey).map((v) => v.pointId),
  );
}

/** Everything the two of them have marked answered, newest first. */
export const useAnswered = () =>
  useQuery({ queryKey: keys.answered, queryFn: prayersApi.answered });

export const useAnswer = () => useWrite(prayerWrites.answer);
export const useUnanswer = () => useWrite(prayerWrites.unanswer);

export const useSavePoints = () => useWrite(prayerWrites.savePoints);
export const usePublish = () => useWrite(prayerWrites.publish);
export const useSaveReflection = () => useWrite(prayerWrites.reflection);
