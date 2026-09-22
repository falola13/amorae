// Facts the Terms and Privacy pages depend on, kept in one place so they're
// updated together. Both documents are drafts until reviewed by a lawyer.
export const legal = {
  /** Shown as "Last updated" on both documents. */
  updated: "22 September 2026",
  /** Where people write about their data or these terms. Set before launch. */
  contactEmail: null as string | null,
  /** How long a sign-in lasts (the API's SESSION_TTL default). */
  sessionDays: 30,
};
