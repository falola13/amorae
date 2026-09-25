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
