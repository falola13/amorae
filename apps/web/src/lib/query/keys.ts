// Query keys in one place so invalidation stays consistent across features.
export const keys = {
  me: ["me"] as const,
  couple: ["couple"] as const,
  endedCouples: ["couple", "ended"] as const,
  week: ["prayers", "current"] as const,
  /** Prefix of every weekById key, for invalidating all of them at once. */
  weeksById: ["prayers", "week"] as const,
  weekById: (id: string) => ["prayers", "week", id] as const,
  history: ["prayers", "history"] as const,
  answered: ["prayers", "answered"] as const,
  events: ["events"] as const,
  event: (id: string) => ["events", id] as const,
  goals: ["goals"] as const,
  goal: (id: string) => ["goals", id] as const,
  /** Prefix of every challenge query: the running ones, templates, past, and each by id. */
  challenge: ["challenge"] as const,
  /** Every running challenge, oldest first. */
  challengeActive: ["challenge", "active"] as const,
  challengeTemplates: ["challenge", "templates"] as const,
  challengePast: ["challenge", "past"] as const,
  /** Prefix of every challengeById key. */
  challengesById: ["challenge", "by"] as const,
  challengeById: (id: string) => ["challenge", "by", id] as const,
  journal: ["journal"] as const,
  appreciations: ["appreciations"] as const,
  memories: ["memories"] as const,
  milestones: ["milestones"] as const,
  prefs: ["notifications", "preferences"] as const,
  sessions: ["sessions"] as const,
  inbox: ["notifications", "inbox"] as const,
  /** Prefix of every timeline filter, for invalidating them all. */
  timelineAll: ["timeline"] as const,
  timeline: (filter: string) => ["timeline", filter] as const,
};
