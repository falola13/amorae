// Mirrors the Go API's JSON contract (snake_case), see docs/API.md. The
// auth and users shapes exist in Go today; the rest is the proposed contract
// the mock adapter implements and the Go modules will follow.

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
  me: User;
  partner: Partner | null;
  invite_code: string;
  started_on?: string;
  /** couple is true whenever this object exists; install and notifications are per person. */
  onboarding: { couple: boolean; install: boolean; notifications: boolean };
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

export interface ChallengeDay {
  n: number;
  text: string;
  done: boolean;
  skipped?: boolean;
}

export interface Challenge {
  id: string;
  title: string;
  started_on: string;
  days: ChallengeDay[];
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
