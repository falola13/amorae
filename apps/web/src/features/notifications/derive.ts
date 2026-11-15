import type { IconName } from "@/components/icons";
import type { NotificationItem, NotificationKind } from "@/lib/api/types";
import { iso, time12 } from "@/lib/dates";

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
  letter: "letter",
  prayer_request: "book",
  prayed_for_you: "check",
  question_answered: "note",
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

/** The nudges your partner sent you today, from the inbox: how many, and the
 *  latest. Day boundaries are your own, as the rest of the inbox's are. */
export function nudgesToday(
  items: NotificationItem[],
  todayIso: string,
): { count: number; last: Date } | null {
  let count = 0;
  let last: Date | null = null;
  for (const item of items) {
    if (item.kind !== "nudge") continue;
    const at = new Date(item.created_at);
    if (Number.isNaN(at.getTime()) || iso(at) !== todayIso) continue;
    count++;
    if (!last || at > last) last = at;
  }
  return last ? { count, last } : null;
}

/** "Ada thought of you · 3:12 pm", or with several, "Ada thought of you 3 times today · last 3:12 pm". */
export function nudgeLine(first: string, count: number, last: Date): string {
  const at = time12(
    `${String(last.getHours()).padStart(2, "0")}:${String(last.getMinutes()).padStart(2, "0")}`,
  );
  return count > 1
    ? `${first} thought of you ${count} times today · last ${at}`
    : `${first} thought of you · ${at}`;
}
