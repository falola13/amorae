"use client";

import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

import { keys } from "@/lib/query/keys";
import type { TimelineFilter } from "@/lib/api/types";
import { timelineApi } from "./api";

export function useTimeline(filter: TimelineFilter, month?: string) {
  return useInfiniteQuery({
    queryKey: keys.timeline(filter, month),
    queryFn: ({ pageParam }: { pageParam?: string }) => timelineApi.page(filter, pageParam, month),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next ?? undefined,
  });
}

/** Which months have entries for this filter, for the "Jump to" sheet. */
export function useTimelineMonths(filter: TimelineFilter, enabled = true) {
  return useQuery({
    queryKey: keys.timelineMonths(filter),
    queryFn: () => timelineApi.months(filter),
    enabled,
  });
}
