import { http } from "@/lib/api/http";
import type { TimelineFilter, TimelinePage } from "@/lib/api/types";

export const timelineApi = {
  page: (filter: TimelineFilter, before?: string) =>
    http
      .get<TimelinePage>("/timeline", { params: { filter, before, limit: 30 } })
      .then((r) => r.data),
};
