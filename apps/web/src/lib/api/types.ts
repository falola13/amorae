// Mirrors the Go API's JSON contract (snake_case, docs/API.md). Auth/users/couples
// exist in Go; the rest is a planned contract the screens were built against.

/** A birthday. Month and day are always known; the year is optional. */
export interface Birthday {
  month: number; // 1-12
  day: number; // 1-31
  year: number | null;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  timezone: string;
  birthday: Birthday | null;
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
  /** When the prayer week turns over — not me.timezone, which is for reminders (DEC-27). */
  timezone: string;
  /** role is your own label in the couple, like the partner's. */
  me: User & { role: string };
  partner: Partner | null;
  invite_code: string;
  started_on?: string;
  /** couple is true whenever this object exists; install and notifications are per person. */
  onboarding: { couple: boolean; install: boolean; notifications: boolean };
}

/** A couple someone used to be in — read-only for 30 days after ending, then deleted. */
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
  /** Weekdays this is prayed, 0 (Sun) to 6 (Sat). Empty means every day. */
  weekdays: number[];
  /** When this was marked answered. Absent until it is, which is what the
   *  answered treatment keys off — there is no boolean. */
  answered_at?: string;
  /** The day it was, in the couple's timezone — the one to show. `answered_at`
   *  is the instant, for ordering; the two can fall on different dates. */
  answered_on?: string;
  /** Who noticed. Answering is for the couple, but it is worth knowing. */
  answered_by?: string;
  /** A line about what happened. Optional even once answered. */
  answer_note?: string;
}

/** An answered prayer, carrying enough of its week to be placed in time. */
export interface AnsweredPrayer extends PrayerPoint {
  week_id: string;
  week_start: string;
}

export type WeekStatus = "draft" | "published" | "waiting";

/** One day of the week, Sunday first. */
export interface PrayerDay {
  date: string; // ISO date
  /** Point ids scheduled for this day. */
  points: string[];
  /** Point ids I prayed this day. */
  mine: string[];
  /** Point ids my partner prayed this day. */
  partner: string[];
}

export interface PrayerWeek {
  id: string;
  week_start: string; // ISO date, a Sunday
  week_end: string;
  /** Whose turn it is to write this week — either of you can still edit or publish it. */
  setter_id: string;
  status: WeekStatus;
  points: PrayerPoint[];
  /** Current week: point ids prayed TODAY, by me/partner. Past weeks: ids prayed on at least one day. */
  my_completed: string[];
  partner_completed: string[];
  /** Couple-local date (YYYY-MM-DD). Present only for the current week. */
  today?: string;
  /** Sun..Sat for this week. */
  days: PrayerDay[];
  /** Point ids I can't edit — my partner already prayed them on some day this week. */
  locked: string[];
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
  /** "together" (both of you) or "mine" (just one of you). */
  kind: "together" | "mine";
  /** Who made it. Absent (null) on events from before this was recorded. */
  created_by: string | null;
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

/** One day of a challenge. `done`/`skipped` are yours; partner_* sit alongside,
 *  and neither side can change the other's (DEC-30). */
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
  /** Signed, unguessable, generated per request. Absent when there is none. */
  photo_url?: string;
}

/** Permission to upload one file, to one name the server chose. */
export interface PhotoTicket {
  upload_url: string;
  /** Send exactly these, plus the file. Rebuilding the list breaks the signature. */
  fields: Record<string, string>;
}

export interface Milestone {
  id: string;
  title: string;
  date: string;
  sub?: string;
  reminder?: boolean;
  /** Set by the server for one it derived itself — the couple's anniversary or a
   *  profile's birthday. Absent (not undefined-checked; just not there) on one a
   *  person added by hand. Optional here only so a hand-added milestone can be
   *  built without it; the API always sends it back. */
  source?: null | "anniversary" | "birthday";
  /** False only for a birthday with no year on file — its `date` then carries
   *  2000 as a placeholder, which is never shown (see milestones.ts's yearsBy). */
  year_known?: boolean;
  /** The user id this is about. Only present on a birthday-sourced milestone;
   *  tells "From your profile" apart from "From {partner}'s profile". */
  about?: string;
}

export interface NotificationPrefs {
  new_week: boolean;
  prayer_reminder: boolean;
  reminder_time: string;
  event_reminders: boolean;
  /** A day or two after, ask how it went. */
  event_followups: boolean;
  /** When your partner adds a together event. */
  partner_events: boolean;
  /** Whether the partner's "thinking of you" nudge may reach you. */
  nudges: boolean;
  /** One of REMINDER_OPTIONS' values (features/together/events.ts); "" is None. */
  default_event_reminder: string;
  important_dates: boolean;
  appreciation: boolean;
  journal: boolean;
  goals: boolean;
  challenges: boolean;
  prayer_answered: boolean;
  /** When the second of you finishes something you are both doing. */
  together: boolean;
  /** A moment kept on this day in an earlier year. */
  memories: boolean;
  /** Halfway and done — not every contribution, which is `goals`. */
  goal_milestones: boolean;
  /** HH:MM in your own zone; both empty means no quiet hours. */
  quiet_from: string;
  quiet_to: string;
  /** Most notifications in one day. 0 means no limit. */
  daily_cap: number;
  /** The largest cap the API will accept; read-only. */
  max_daily_cap: number;
}

/** Every kind of notification the API can put in the inbox. */
export type NotificationKind =
  | "new_week"
  | "week_published"
  | "prayer_reminder"
  | "event_reminder"
  | "event_added"
  | "event_over"
  | "important_date"
  | "appreciation"
  | "journal"
  | "goal"
  | "goal_milestone"
  | "goal_milestones"
  | "goal_crossing"
  | "challenge"
  | "prayer_answered"
  | "both_prayed"
  | "both_marked"
  | "memory_on_this_day"
  | "nudge";

/** One row of the notifications inbox (GET /v1/notifications/inbox). Newest first,
 *  at most 30 — the server keeps only that many. */
export interface NotificationItem {
  id: string;
  kind: NotificationKind;
  title: string;
  body: string;
  /** Where tapping it goes, an in-app path. */
  path: string;
  created_at: string;
  read: boolean;
}

export interface NotificationsInbox {
  items: NotificationItem[];
  unread: number;
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
