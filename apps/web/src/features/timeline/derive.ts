import { monthLabel } from "@/lib/dates";
import type { TimelineItem } from "@/lib/api/types";

export interface TimelineGroup {
  month: string;
  items: TimelineItem[];
}

/** Groups already-ordered (newest first) items by their month header, e.g.
 *  "September 2026". Consecutive items in the same month join one group,
 *  so a month that spans two pages still gets one header when the items
 *  arrive back to back. */
export function groupByMonth(items: TimelineItem[]): TimelineGroup[] {
  const groups: TimelineGroup[] = [];
  for (const item of items) {
    const month = monthLabel(item.date);
    const last = groups[groups.length - 1];
    if (last && last.month === month) last.items.push(item);
    else groups.push({ month, items: [item] });
  }
  return groups;
}
