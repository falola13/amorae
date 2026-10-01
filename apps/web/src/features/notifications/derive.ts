import type { IconName } from "@/components/icons";
import type { NotificationItem, NotificationKind } from "@/lib/api/types";
import { iso } from "@/lib/dates";

/** Bell as the fallback: any kind the client doesn't recognise yet still shows something. */
const ICON_BY_KIND: Record<NotificationKind, IconName> = {
  new_week: "book",
  week_published: "book",
  prayer_reminder: "bell",
  event_reminder: "calendar",
  event_added: "calendar",
  event_over: "check",
  important_date: "gift",
  appreciation: "heart",
  journal: "note",
  goal: "target",
  goal_milestone: "flag",
  goal_milestones: "flag",
  goal_crossing: "flag",
  challenge: "sliders",
  challenge_started: "flag",
  prayer_answered: "check",
  both_prayed: "users",
  both_marked: "check",
  memory_on_this_day: "image",
  nudge: "together",
};

export const iconForKind = (kind: NotificationKind): IconName => ICON_BY_KIND[kind] ?? "bell";

/** Today's items first, then everything earlier — each newest first, since the
 *  API already sends the list that way. */
export function groupInbox(
  items: NotificationItem[],
  todayIso: string,
): { today: NotificationItem[]; earlier: NotificationItem[] } {
  const today: NotificationItem[] = [];
  const earlier: NotificationItem[] = [];
  for (const item of items) {
    (iso(new Date(item.created_at)) === todayIso ? today : earlier).push(item);
  }
  return { today, earlier };
}
