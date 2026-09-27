"use client";

import { useInfiniteQuery } from "@tanstack/react-query";

import { keys } from "@/lib/query/keys";
import type { TimelineFilter } from "@/lib/api/types";
import { timelineApi } from "./api";

export function useTimeline(filter: TimelineFilter) {
  return useInfiniteQuery({
    queryKey: keys.timeline(filter),
    queryFn: ({ pageParam }: { pageParam?: string }) => timelineApi.page(filter, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next ?? undefined,
  });
}
