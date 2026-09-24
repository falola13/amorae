// Mirrors the Go API's JSON contract (snake_case), see docs/API.md. The
// auth, users and couples shapes exist in Go; the rest is a planned contract
// the screens were built against. When a Go module lands with a different
// shape, change it here and in docs/API.md in the same pull request.

export interface User {
  id: string;
  email: string;
  display_name: string;
  timezone: string;
  created_at: string;
  updated_at: string;
}

export interface AuthResult {
  token: string;
  expires_at: string;
  user: User;
}

export interface Partner {
  id: string;
  display_name: string;
  /** How they label themselves in the couple ("Husband"), not a permission. */
  role: string;
}

export interface Couple {
  id: string;
  name: string;
  /**
   * When this couple's prayer week turns over. Not the same as me.timezone,
   * which is when your own reminders fire: partners in different places must
   * still agree on which week it is (DEC-27).
   */
  timezone: string;
  /** role is your own label in the couple, like the partner's. */
  me: User & { role: string };
  partner: Partner | null;
  invite_code: string;
  started_on?: string;
  /** couple is true whenever this object exists; install and notifications are per person. */
  onboarding: { couple: boolean; install: boolean; notifications: boolean };
}

/**
 * A couple someone used to be in. Ending a couple ends it for both partners;
 * for 30 days afterwards it can still be read and downloaded, and then it is
 * deleted. There is nothing to act on here — no invite, no onboarding.
 */
export interface EndedCouple {
  id: string;
  name: string;
  people: Partner[];
  started_on?: string;
  dissolved_at: string;
  read_only_until: string;
}

export interface PrayerPoint {
  id: string;
  title: string;
  text: string;
  scripture?: string;
  verse?: string;
  position: number;
}

export type WeekStatus = "draft" | "published" | "waiting";

export interface PrayerWeek {
  id: string;
  week_start: string; // ISO date, a Sunday
  week_end: string;
  setter_id: string;
  status: WeekStatus;
  points: PrayerPoint[];
  my_completed: string[];
  partner_completed: string[];
  reflection?: string;
}

export interface ChecklistItem {
  id: string;
  text: string;
  done: boolean;
}

export interface Event {
  id: string;
  title: string;
  date: string;
  start_time?: string;
  end_time?: string;
  location?: string;
  reminder?: string;
  notes?: string;
  checklist: ChecklistItem[];
  done: boolean;
}

export interface GoalProgress {
  id: string;
  user_id: string;
  amount: number;
  date: string;
}

export interface Goal {
  id: string;
  title: string;
  why?: string;
  target: number;
  unit: "naira" | "count";
  unit_label?: string;
  progress: GoalProgress[];
  start_date: string;
  end_date: string;
  done: boolean;
}

/**
 * One day of a challenge. `done` and `skipped` are YOURS, exactly as
 * `my_completed` is on a prayer week; your partner's sit alongside so both of
 * you can see both, and neither can change the other's (DEC-30).
 */
export interface ChallengeDay {
  n: number;
  text: string;
  done: boolean;
  skipped?: boolean;
  partner_done?: boolean;
  partner_skipped?: boolean;
}

export interface Challenge {
  id: string;
  template: string;
  title: string;
  started_on: string;
  days: ChallengeDay[];
}

/** One of the curated challenges a couple can start. */
export interface ChallengeTemplate {
  key: string;
  title: string;
  blurb: string;
  days: number;
}

export type JournalTag = "Gratitude" | "Reflection" | "Memory" | "Appreciation" | "Plans";

export interface JournalEntry {
  id: string;
  author_id: string;
  date: string;
  tag: JournalTag;
  text: string;
}

export interface Appreciation {
  id: string;
  from_id: string;
  date: string;
  text: string;
}

export interface Memory {
  id: string;
  title: string;
  date: string;
  location?: string;
  note?: string;
  has_photo: boolean;
}

export interface Milestone {
  id: string;
  title: string;
  date: string;
  sub?: string;
  reminder?: boolean;
}

export interface NotificationPrefs {
  new_week: boolean;
  prayer_reminder: boolean;
  reminder_time: string;
  event_reminders: boolean;
  important_dates: boolean;
  appreciation: boolean;
  journal: boolean;
  goals: boolean;
  challenges: boolean;
}

/** One row of "where you're signed in" (GET /v1/sessions), for your own account. */
export interface SessionInfo {
  /** The session making this request. */
  current: boolean;
  /** A coarse label the API builds from the user agent, e.g. "Safari on iPhone". */
  device: string;
  created_at: string;
  /** Absent until the session is used again after sign-in. */
  last_used_at?: string;
  expires_at: string;
}
