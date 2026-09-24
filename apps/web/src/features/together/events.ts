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

/**
 * Reminders were free text, typed into an invisible field, which meant
 * nothing could ever act on them and most people got nothing at all. They are
 * a choice from this list now, and the stored value is still the phrase
 * itself — `events.reminder` stays TEXT, and a reminder written before this
 * existed still reads back exactly as it was written.
 *
 * When delivery is built, each of these maps to an offset from the start
 * time. Until then this is the honest half: a fixed set is parseable later,
 * free text never was.
 */
export const eventReminderLabel = (value?: string | null): string => {
  if (!value) return "None";
  return REMINDER_OPTIONS.find((o) => o.value === value)?.label ?? value;
};
