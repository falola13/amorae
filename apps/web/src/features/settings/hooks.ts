"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { NotificationPrefs } from "@/lib/api/types";
import { keys } from "@/lib/query/keys";
import { settingsApi } from "./api";
import { settingsWrites } from "./writes";

export const usePrefs = () => useQuery({ queryKey: keys.prefs, queryFn: settingsApi.prefs });

/** Optimistic: the switch flips at once. Offline, the change waits and survives restarts. */
export function useSavePrefs() {
  const qc = useQueryClient();
  const def = settingsWrites.savePrefs;
  return useMutation({
    mutationKey: def.mutationKey,
    mutationFn: def.mutationFn,
    scope: { id: def.scope! },
    onMutate: async (patch) => {
      await qc.cancelQueries({ queryKey: keys.prefs });
      const prev = qc.getQueryData<NotificationPrefs>(keys.prefs);
      if (prev) qc.setQueryData(keys.prefs, { ...prev, ...patch });
      return { prev };
    },
    onError: (_e, _v, ctx) => { if (ctx?.prev) qc.setQueryData(keys.prefs, ctx.prev); },
    onSettled: () => qc.invalidateQueries({ queryKey: keys.prefs }),
  });
}
