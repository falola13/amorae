"use client";

import { useQuery } from "@tanstack/react-query";

import { keys } from "@/lib/query/keys";
import { useWrite } from "@/lib/query/mutations";
import { notificationsApi } from "./api";
import { notificationsWrites } from "./writes";

// Lightly polled: a focus refetch (React Query's default) plus every 60s while
// the tab is visible, so the bell's count stays close to right without a socket.
export const useInbox = () =>
  useQuery({
    queryKey: keys.inbox,
    queryFn: notificationsApi.inbox,
    refetchInterval: 60_000,
    refetchIntervalInBackground: false,
  });

export const useMarkAllRead = () => useWrite(notificationsWrites.markAllRead);
