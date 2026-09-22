// Mock data store for the front end while the Go modules are being written.
// Lives in localStorage so the app survives reloads and works offline. The
// shapes are the API contract in lib/api/types.ts, so nothing above the
// axios adapter changes when the real endpoints land.
import type { Appreciation, Challenge, Couple, Event, Goal, JournalEntry, Memory, Milestone, NotificationPrefs, PrayerPoint, PrayerWeek } from "../types";
import { addDays, iso, startOfWeek } from "@/lib/dates";
import { MOCK_TODAY } from "./clock";

export const ME = "u-femi";
export const PARTNER = "u-adeola";

/** "Today" is anchored so the demo data is coherent (see ./clock.ts). */
export const TODAY = MOCK_TODAY;
export const TODAY_ISO = iso(TODAY);

export interface MockDB {
  couple: Couple;
  weeks: PrayerWeek[];
  events: Event[];
  goals: Goal[];
  challenge: Challenge;
  journal: JournalEntry[];
  appreciations: Appreciation[];
  memories: Memory[];
  milestones: Milestone[];
  prefs: NotificationPrefs;
}

// Bump when the seed changes shape (v4: couple gained name, partner gained
// role), so browsers that already hold a mock store re-seed instead of
// keeping stale data.
const KEY = "amorae:mock:v4";
export const uid = () => Math.random().toString(36).slice(2, 10);
const wk = (n: number) => iso(addDays(startOfWeek(TODAY), n * 7));
const wkEnd = (n: number) => iso(addDays(startOfWeek(TODAY), n * 7 + 6));

export const pts = (rows: [string, string, string?, string?][]): PrayerPoint[] =>
  rows.map(([title, text, scripture, verse], i) => ({ id: uid(), title, text, scripture, verse, position: i }));

const now = new Date().toISOString();

export function seed(): MockDB {
  const current = pts([
    ["Career direction", "Give us clarity for the next season of our careers, and peace about where it leads.", "Proverbs 3:5-6", "Trust in the Lord with all thine heart; and lean not unto thine own understanding. In all thy ways acknowledge him, and he shall direct thy paths."],
    ["Family", "Keep both our families in peace and good health, and draw us closer to them this year."],
    ["Our relationship", "Teach us patience with each other, and give us the courage for honest conversations.", "Ecclesiastes 4:9-10", "Two are better than one; because they have a good reward for their labour. For if they fall, the one will lift up his fellow."],
    ["Wisdom", "Give us wisdom for the decisions ahead of us, and the patience to make them together.", "James 1:5", "If any of you lack wisdom, let him ask of God, that giveth to all men liberally, and upbraideth not; and it shall be given him."],
    ["Future plans", "Help us hold our plans with open hands, and trust you with the timing.", "Proverbs 16:9", "A man’s heart deviseth his way: but the Lord directeth his steps."],
  ]);
  const last = pts([["A new job for Adeola", "Open the right door, and give her peace while she waits."], ["Our parents", "Health and joy for both sets of parents."], ["Generosity", "Make us people who give without keeping score."], ["Friends who are struggling", "For T and for Kemi. Carry them through this month."], ["Gratitude", "For a year we did not expect and would not trade."]]);
  const w2 = pts([["Patience", ""], ["Church", ""], ["Health", ""], ["Provision", ""], ["Joy", ""], ["Rest", ""]]);
  const w3 = pts([["Trust", ""], ["Work", ""], ["Peace", ""], ["Our home", ""]]);
  return {
    couple: {
      id: "c1",
      name: "Femi & Adeola",
      me: { id: ME, display_name: "Femi", email: "femi@example.com", timezone: "Africa/Lagos", created_at: now, updated_at: now },
      partner: { id: PARTNER, display_name: "Adeola", role: "partner" },
      // Same shape the Go API returns: three letters, three digits, shown as ABC-123.
      invite_code: "KQF-729",
      started_on: "2024-02-12",
      onboarding: { couple: true, install: true, notifications: true },
    },
    weeks: [
      { id: "w0", week_start: wk(0), week_end: wkEnd(0), setter_id: PARTNER, status: "published", points: current, my_completed: current.slice(0, 3).map((p) => p.id), partner_completed: current.slice(0, 4).map((p) => p.id) },
      { id: "w-1", week_start: wk(-1), week_end: wkEnd(-1), setter_id: ME, status: "published", points: last, my_completed: [last[0].id, last[1].id, last[2].id, last[4].id], partner_completed: last.map((p) => p.id), reflection: "We talked more honestly this week than we have in a while." },
      { id: "w-2", week_start: wk(-2), week_end: wkEnd(-2), setter_id: PARTNER, status: "published", points: w2, my_completed: w2.map((p) => p.id), partner_completed: w2.map((p) => p.id) },
      { id: "w-3", week_start: wk(-3), week_end: wkEnd(-3), setter_id: PARTNER, status: "published", points: w3, my_completed: w3.map((p) => p.id), partner_completed: w3.slice(0, 3).map((p) => p.id) },
    ],
    events: [
      { id: "e1", title: "Dinner together", date: TODAY_ISO, start_time: "19:00", end_time: "21:00", location: "At home, we’re cooking", reminder: "1 hour before", checklist: [], done: false },
      { id: "e2", title: "Date night", date: iso(addDays(TODAY, 4)), start_time: "18:30", end_time: "21:00", location: "The rooftop place in Ikoyi", reminder: "1 hour before", notes: "Phones away once we sit down.", checklist: [{ id: "c1", text: "Book the table", done: true }, { id: "c2", text: "Arrange a ride home", done: false }], done: false },
      { id: "e3", title: "Sunday service", date: wk(1), start_time: "09:00", end_time: "11:30", location: "Together", reminder: "1 hour before", checklist: [], done: false },
      { id: "e0", title: "Movie night", date: iso(addDays(TODAY, -8)), start_time: "20:00", checklist: [], done: true },
    ],
    goals: [
      { id: "g1", title: "Save ₦500,000 together", why: "So December feels generous instead of tight.", target: 500000, unit: "naira", start_date: "2026-09-01", end_date: "2026-12-31", done: false, progress: [{ id: "p1", user_id: PARTNER, amount: 50000, date: "2026-09-12" }, { id: "p2", user_id: ME, amount: 60000, date: "2026-09-19" }, { id: "p3", user_id: PARTNER, amount: 40000, date: "2026-09-26" }, { id: "p0", user_id: ME, amount: 160000, date: "2026-09-05" }] },
      { id: "g2", title: "Read one book this month", target: 14, unit: "count", unit_label: "chapters", start_date: "2026-10-01", end_date: "2026-10-31", done: false, progress: [{ id: "p4", user_id: ME, amount: 6, date: "2026-09-27" }] },
      { id: "g3", title: "Plan our December trip", target: 1, unit: "count", start_date: "2026-08-01", end_date: "2026-09-15", done: true, progress: [{ id: "p5", user_id: PARTNER, amount: 1, date: "2026-09-10" }] },
    ],
    challenge: {
      id: "ch1", title: "7-day connection", started_on: iso(addDays(TODAY, -2)),
      days: [
        { n: 1, text: "Tell your partner something you appreciate", done: true }, { n: 2, text: "Have dinner without phones", done: true }, { n: 3, text: "Pray together", done: false },
        { n: 4, text: "Talk about one future goal", done: false }, { n: 5, text: "Share a favourite memory", done: false }, { n: 6, text: "Take a walk together", done: false }, { n: 7, text: "Plan something to look forward to", done: false },
      ],
    },
    journal: [
      { id: "j1", author_id: PARTNER, date: wk(0), tag: "Gratitude", text: "Grateful for a slow Sunday. We didn’t do much and it was exactly right." },
      { id: "j2", author_id: ME, date: iso(addDays(TODAY, -5)), tag: "Reflection", text: "We handled the money conversation better than last time. We listened first." },
      { id: "j3", author_id: PARTNER, date: "2026-09-21", tag: "Memory", text: "Movie night. You fell asleep before the good part, again." },
    ],
    appreciations: [
      { id: "a1", from_id: PARTNER, date: wk(0), text: "I appreciate how you always remember the small things." },
      { id: "a2", from_id: ME, date: "2026-09-22", text: "I appreciate you praying for my interview before I even asked." },
    ],
    memories: [
      { id: "m1", title: "Our first Amorae date night", date: "2026-09-21", location: "Lekki", note: "We stayed until they stacked the chairs.", has_photo: true },
      { id: "m2", title: "Started our 30-day prayer journey", date: "2026-09-28", has_photo: true },
    ],
    milestones: [
      { id: "d1", title: "Adeola’s birthday", date: "2026-11-14", sub: "Reminder a week before", reminder: true },
      { id: "d2", title: "Our anniversary", date: "2027-02-12", sub: "Three years together", reminder: true },
      { id: "d3", title: "Started our 30-day prayer journey", date: "2026-09-28" },
      { id: "d4", title: "The day we started", date: "2024-02-12" },
    ],
    prefs: { new_week: true, prayer_reminder: true, reminder_time: "21:00", event_reminders: true, important_dates: true, appreciation: true, journal: true, goals: false, challenges: false },
  };
}

let cache: MockDB | null = null;

export function load(): MockDB {
  if (cache) return cache;
  if (typeof window === "undefined") return seed();
  try {
    const raw = localStorage.getItem(KEY);
    cache = raw ? (JSON.parse(raw) as MockDB) : seed();
  } catch {
    cache = seed();
  }
  return cache;
}

export function save(db: MockDB) {
  cache = db;
  try {
    localStorage.setItem(KEY, JSON.stringify(db));
  } catch {
    /* private mode: keep in memory */
  }
}

export function reset() {
  cache = null;
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* noop */
  }
}
