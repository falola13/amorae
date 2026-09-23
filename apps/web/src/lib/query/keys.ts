// Query keys in one place so invalidation stays consistent across features.
export const keys = {
  me: ["me"] as const,
  couple: ["couple"] as const,
  week: ["prayers", "current"] as const,
  /** Prefix of every weekById key, for invalidating all of them at once. */
  weeksById: ["prayers", "week"] as const,
  weekById: (id: string) => ["prayers", "week", id] as const,
  history: ["prayers", "history"] as const,
  events: ["events"] as const,
  event: (id: string) => ["events", id] as const,
  goals: ["goals"] as const,
  goal: (id: string) => ["goals", id] as const,
  challenge: ["challenge"] as const,
  journal: ["journal"] as const,
  appreciations: ["appreciations"] as const,
  memories: ["memories"] as const,
  milestones: ["milestones"] as const,
  prefs: ["notifications", "preferences"] as const,
  sessions: ["sessions"] as const,
};
