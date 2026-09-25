import type { Event } from "@/lib/api/types";

/** What an event's reminder can say. */
export const REMINDER_OPTIONS = [
  { value: "", label: "None" },
  { value: "at the time", label: "At the time" },
  { value: "10 minutes before", label: "10 minutes before" },
  { value: "30 minutes before", label: "30 minutes before" },
  { value: "1 hour before", label: "1 hour before" },
  { value: "2 hours before", label: "2 hours before" },
  { value: "the morning of", label: "The morning of" },
  { value: "1 day before", label: "The day before" },
] as const;

// `events.reminder` stays free-form TEXT so old entries still read back unchanged; each option here
// will map to a time offset once delivery is built.
export const eventReminderLabel = (value?: string | null): string => {
  if (!value) return "None";
  return REMINDER_OPTIONS.find((o) => o.value === value)?.label ?? value;
};

type OwnedEvent = Pick<Event, "kind" | "created_by">;

/** Can this person edit, delete, complete, or tick off this event? Either of
 *  you, for a together one. For a "mine" one, only whoever made it — unless
 *  that's unknown (an event from before this was recorded), which allows it
 *  rather than locking someone out of their own plan. */
export function canManageEvent(event: OwnedEvent, meId?: string): boolean {
  if (event.kind !== "mine") return true;
  if (!event.created_by) return true;
  return event.created_by === meId;
}

/** Only the creator gets to change who an event is for. Unknown creator
 *  (an old event) allows it, same reasoning as canManageEvent. */
export function canChangeEventKind(event: Pick<Event, "created_by">, meId?: string): boolean {
  if (!event.created_by) return true;
  return event.created_by === meId;
}

/** For a "mine" event: whose it is, as a short possessive ("Yours" / "Adeola's").
 *  null for a together event, or one with no recorded owner. */
export function eventOwnerLabel(
  event: OwnedEvent,
  meId?: string,
  partnerName?: string,
): string | null {
  if (event.kind !== "mine" || !event.created_by) return null;
  if (event.created_by === meId) return "Yours";
  return `${partnerName ?? "Their"}’s`;
}

/** For a together event: who added it ("Added by you" / "Added by Adeola").
 *  null for a "mine" event, or one with no recorded creator. */
export function eventAddedByLabel(
  event: OwnedEvent,
  meId?: string,
  partnerName?: string,
): string | null {
  if (event.kind === "mine" || !event.created_by) return null;
  if (event.created_by === meId) return "Added by you";
  return `Added by ${partnerName ?? "your partner"}`;
}
