export type User = { id: string; name: string; email: string; timezone: string };
export type Partner = { id: string; name: string };
export type Couple = { id: string; me: User; partner: Partner | null; inviteCode: string; startedOn?: string };

export type PrayerPoint = { id: string; title: string; text: string; scripture?: string; verse?: string; position: number };
export type WeekStatus = 'draft' | 'published' | 'waiting';
export type PrayerWeek = {
  id: string;
  start: string; // ISO date, a Sunday
  end: string;
  setterId: string;
  status: WeekStatus;
  points: PrayerPoint[];
  myCompleted: string[];
  partnerCompleted: string[];
  reflection?: string;
};

export type Event = { id: string; title: string; date: string; start?: string; end?: string; location?: string; reminder?: string; notes?: string; checklist: { id: string; text: string; done: boolean }[]; done: boolean };
export type Goal = { id: string; title: string; why?: string; target: number; unit: 'naira' | 'count'; unitLabel?: string; progress: { id: string; by: string; amount: number; date: string }[]; start: string; end: string; done: boolean };
export type Challenge = { id: string; title: string; days: { n: number; text: string; done: boolean; skipped?: boolean }[]; startedOn: string };
export type JournalEntry = { id: string; authorId: string; date: string; tag: 'Gratitude' | 'Reflection' | 'Memory' | 'Appreciation' | 'Plans'; text: string };
export type Appreciation = { id: string; fromId: string; date: string; text: string };
export type Memory = { id: string; title: string; date: string; location?: string; note?: string; hasPhoto: boolean };
export type Milestone = { id: string; title: string; date: string; sub?: string; reminder?: boolean };

export type NotificationPrefs = {
  newWeek: boolean; prayerReminder: boolean; reminderTime: string; eventReminders: boolean; importantDates: boolean;
  appreciation: boolean; journal: boolean; goals: boolean; challenges: boolean;
};
