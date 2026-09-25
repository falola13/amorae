/** Used by two distinct settings (DEC-27): your own zone for reminders, and the
 *  couple's shared zone for when the prayer week turns over. */
export const ZONES = [
  "Africa/Lagos",
  "America/New_York",
  "Europe/London",
  "Africa/Nairobi",
  "Asia/Dubai",
  "UTC",
] as const;

/** "America/New York" — the IANA name, easier to read. Optional value degrades
 *  to a blank label rather than crashing on a missing/partial field. */
export const zoneLabel = (zone?: string | null) => (zone ? zone.replace(/_/g, " ") : "");
