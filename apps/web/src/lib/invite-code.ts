const CHARS = 6;

/** The 6 letters/digits only. Hyphens and spaces are dropped. */
export function inviteChars(raw: string): string {
  return raw
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, "")
    .slice(0, CHARS);
}

/** Display and input shape: ABC123 → ABC-123. Still only 6 characters. */
export function formatInviteCode(raw: string): string {
  const chars = inviteChars(raw);
  if (chars.length <= 3) return chars;
  return `${chars.slice(0, 3)}-${chars.slice(3)}`;
}
