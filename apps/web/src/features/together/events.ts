import type { Event } from "@/lib/api/types";
import { time12 } from "@/lib/dates";

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

/** The most reminders one event can carry. */
export const MAX_REMINDERS = 3;

/** The "at HH:MM" form: a set time on the event's day. */
export const AT_TIME = /^at (\d{2}:\d{2})$/;

export const atTimeReminder = (time: string): string => `at ${time}`;
export const reminderTime = (value: string): string | null => AT_TIME.exec(value)?.[1] ?? null;

/** A reminder as it reads on screen: "1 hour before" -> "1 hour before" capitalised, "at 16:00" -> "At 4:00 pm". */
export const eventReminderLabel = (value?: string | null): string => {
  if (!value) return "None";
  const known = REMINDER_OPTIONS.find((o) => o.value === value)?.label;
  if (known) return known;
  const t = reminderTime(value);
  if (t) return `At ${time12(t)}`;
  return value.charAt(0).toUpperCase() + value.slice(1);
};

/** Several reminders in one line: the first, or "{n} reminders". Empty when there are none. */
export const remindersSummary = (reminders: readonly string[]): string =>
  reminders.length === 0
    ? ""
    : reminders.length === 1
      ? eventReminderLabel(reminders[0])
      : `${reminders.length} reminders`;

export type EventPhase = "upcoming" | "ongoing" | "over";

type Timed = Pick<Event, "date" | "start_time" | "end_time" | "done" | "didnt_happen">;

const at = (date: string, time: string, plusDays = 0): Date => {
  const [y, mo, d] = date.split("-").map(Number);
  const [h, mi] = time.split(":").map(Number);
  return new Date(y, mo - 1, d + plusDays, h || 0, mi || 0, 0, 0);
};

/** When an event starts and ends, from its date and optional times.
 *  Start: the start time, or midnight. End: the end time on that day (the next day if it isn't after the start);
 *  else two hours after a start time; else the end of the day. */
export function eventWindow(e: Pick<Event, "date" | "start_time" | "end_time">): {
  start: Date;
  end: Date;
} {
  const start = at(e.date, e.start_time || "00:00");
  let end: Date;
  if (e.end_time) {
    end = at(e.date, e.end_time);
    if (end <= start) end = at(e.date, e.end_time, 1);
  } else if (e.start_time) {
    end = new Date(start.getTime() + 2 * 60 * 60 * 1000);
  } else {
    end = at(e.date, "23:59");
    end.setSeconds(59);
  }
  return { start, end };
}

/** The one rule for where an event is right now. A done or didn't-happen event is always "over". */
export function eventPhase(e: Timed, now: Date): EventPhase {
  if (e.done || e.didnt_happen) return "over";
  const { start, end } = eventWindow(e);
  if (now < start) return "upcoming";
  if (now < end) return "ongoing";
  return "over";
}

/** "until 9:00 pm", or "" when no end time was set. */
export const endsLabel = (e: Pick<Event, "end_time">): string =>
  e.end_time ? `until ${time12(e.end_time)}` : "";

/** "Now · until 9:00 pm", or just "Now" with no end time. */
export const nowLabel = (e: Pick<Event, "end_time">): string =>
  e.end_time ? `Now · ${endsLabel(e)}` : "Now";

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
