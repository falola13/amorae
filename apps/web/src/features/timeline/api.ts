import { http } from "@/lib/api/http";
import type { TimelineFilter, TimelineMonth, TimelinePage } from "@/lib/api/types";

export const timelineApi = {
  // `month` only starts the first page; later pages carry the cursor instead.
  page: (filter: TimelineFilter, before?: string, month?: string) =>
    http
      .get<TimelinePage>("/timeline", {
        params: { filter, before, month: before ? undefined : month, limit: 30 },
      })
      .then((r) => r.data),
  months: (filter: TimelineFilter) =>
    http.get<TimelineMonth[]>("/timeline/months", { params: { filter } }).then((r) => r.data),
};
