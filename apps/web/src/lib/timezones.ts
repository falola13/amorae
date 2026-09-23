/**
 * The zones Amorae offers. Two different settings use this list, and they are
 * not the same thing (DEC-27):
 *
 * - your own zone decides when your reminders fire;
 * - the couple's zone decides when the prayer week turns over, so it is one
 *   setting shared by both partners rather than one each.
 */
export const ZONES = [
  "Africa/Lagos",
  "America/New_York",
  "Europe/London",
  "Africa/Nairobi",
  "Asia/Dubai",
  "UTC",
] as const;

/** "America/New York" — the IANA name, just easier to read. */
export const zoneLabel = (zone: string) => zone.replace(/_/g, " ");
