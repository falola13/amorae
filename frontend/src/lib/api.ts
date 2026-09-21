/*
  Mock API. Same shape the Go backend will expose, so swapping is a matter of
  replacing the function bodies with fetch() calls to /prayers, /events, /goals…
  State lives in localStorage so the app survives reloads and works offline.
  Mutations that fail while offline are queued and replayed when we're back.
*/
import type { Appreciation, Challenge, Couple, Event, Goal, JournalEntry, Memory, Milestone, NotificationPrefs, PrayerPoint, PrayerWeek } from './types';
import { addDays, iso, startOfWeek } from './dates';

export const ME = 'u-femi';
export const PARTNER = 'u-adeola';

type DB = {
  session: boolean;
  onboarded: { couple: boolean; install: boolean; notifications: boolean };
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
};

const KEY = 'amorae:db:v1';
const QUEUE_KEY = 'amorae:queue:v1';
const uid = () => Math.random().toString(36).slice(2, 10);
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/** "Today" is anchored so the demo data is coherent. Swap for `new Date()` once the API is live. */
export const TODAY = new Date(2026, 8, 29, 19, 10); // Tuesday 29 September 2026, evening
const todayIso = iso(TODAY);
const thisSunday = iso(startOfWeek(TODAY));
const nextSunday = iso(addDays(startOfWeek(TODAY), 7));
const wk = (n: number) => iso(addDays(startOfWeek(TODAY), n * 7));

const pts = (rows: [string, string, string?, string?][]): PrayerPoint[] => rows.map(([title, text, scripture, verse], i) => ({ id: uid(), title, text, scripture, verse, position: i }));

function seed(): DB {
  const current = pts([
    ['Career direction', 'Give us clarity for the next season of our careers, and peace about where it leads.', 'Proverbs 3:5-6', 'Trust in the Lord with all thine heart; and lean not unto thine own understanding. In all thy ways acknowledge him, and he shall direct thy paths.'],
    ['Family', 'Keep both our families in peace and good health, and draw us closer to them this year.'],
    ['Our relationship', 'Teach us patience with each other, and give us the courage for honest conversations.', 'Ecclesiastes 4:9-10', 'Two are better than one; because they have a good reward for their labour. For if they fall, the one will lift up his fellow.'],
    ['Wisdom', 'Give us wisdom for the decisions ahead of us, and the patience to make them together.', 'James 1:5', 'If any of you lack wisdom, let him ask of God, that giveth to all men liberally, and upbraideth not; and it shall be given him.'],
    ['Future plans', 'Help us hold our plans with open hands, and trust you with the timing.', 'Proverbs 16:9', 'A man’s heart deviseth his way: but the Lord directeth his steps.'],
  ]);
  const last = pts([['A new job for Adeola', 'Open the right door, and give her peace while she waits.'], ['Our parents', 'Health and joy for both sets of parents.'], ['Generosity', 'Make us people who give without keeping score.'], ['Friends who are struggling', 'For T and for Kemi. Carry them through this month.'], ['Gratitude', 'For a year we did not expect and would not trade.']]);
  const w2 = pts([['Patience', ''], ['Church', ''], ['Health', ''], ['Provision', ''], ['Joy', ''], ['Rest', '']]);
  const w3 = pts([['Trust', ''], ['Work', ''], ['Peace', ''], ['Our home', '']]);
  return {
    session: false,
    onboarded: { couple: false, install: false, notifications: false },
    couple: { id: 'c1', me: { id: ME, name: 'Femi', email: 'femi@example.com', timezone: 'Africa/Lagos' }, partner: { id: PARTNER, name: 'Adeola' }, inviteCode: 'K7QF-2M9D', startedOn: '2024-02-12' },
    weeks: [
      { id: 'w0', start: thisSunday, end: iso(addDays(startOfWeek(TODAY), 6)), setterId: PARTNER, status: 'published', points: current, myCompleted: current.slice(0, 3).map((p) => p.id), partnerCompleted: current.slice(0, 4).map((p) => p.id) },
      { id: 'w-1', start: wk(-1), end: iso(addDays(startOfWeek(TODAY), -1)), setterId: ME, status: 'published', points: last, myCompleted: [last[0].id, last[1].id, last[2].id, last[4].id], partnerCompleted: last.map((p) => p.id), reflection: 'We talked more honestly this week than we have in a while.' },
      { id: 'w-2', start: wk(-2), end: iso(addDays(startOfWeek(TODAY), -8)), setterId: PARTNER, status: 'published', points: w2, myCompleted: w2.map((p) => p.id), partnerCompleted: w2.map((p) => p.id) },
      { id: 'w-3', start: wk(-3), end: iso(addDays(startOfWeek(TODAY), -15)), setterId: PARTNER, status: 'published', points: w3, myCompleted: w3.map((p) => p.id), partnerCompleted: w3.slice(0, 3).map((p) => p.id) },
    ],
    events: [
      { id: 'e1', title: 'Dinner together', date: todayIso, start: '19:00', end: '21:00', location: 'At home, we’re cooking', reminder: '1 hour before', notes: '', checklist: [], done: false },
      { id: 'e2', title: 'Date night', date: iso(addDays(TODAY, 4)), start: '18:30', end: '21:00', location: 'The rooftop place in Ikoyi', reminder: '1 hour before', notes: 'Phones away once we sit down.', checklist: [{ id: 'c1', text: 'Book the table', done: true }, { id: 'c2', text: 'Arrange a ride home', done: false }], done: false },
      { id: 'e3', title: 'Sunday service', date: nextSunday, start: '09:00', end: '11:30', location: 'Together', reminder: '1 hour before', checklist: [], done: false },
      { id: 'e0', title: 'Movie night', date: iso(addDays(TODAY, -8)), start: '20:00', checklist: [], done: true },
    ],
    goals: [
      { id: 'g1', title: 'Save ₦500,000 together', why: 'So December feels generous instead of tight.', target: 500000, unit: 'naira', start: '2026-09-01', end: '2026-12-31', done: false, progress: [{ id: 'p1', by: PARTNER, amount: 50000, date: '2026-09-12' }, { id: 'p2', by: ME, amount: 60000, date: '2026-09-19' }, { id: 'p3', by: PARTNER, amount: 40000, date: '2026-09-26' }, { id: 'p0', by: ME, amount: 160000, date: '2026-09-05' }] },
      { id: 'g2', title: 'Read one book this month', why: '', target: 14, unit: 'count', unitLabel: 'chapters', start: '2026-10-01', end: '2026-10-31', done: false, progress: [{ id: 'p4', by: ME, amount: 6, date: '2026-09-27' }] },
      { id: 'g3', title: 'Plan our December trip', target: 1, unit: 'count', start: '2026-08-01', end: '2026-09-15', done: true, progress: [{ id: 'p5', by: PARTNER, amount: 1, date: '2026-09-10' }] },
    ],
    challenge: { id: 'ch1', title: '7-day connection', startedOn: iso(addDays(TODAY, -2)), days: [
      { n: 1, text: 'Tell your partner something you appreciate', done: true }, { n: 2, text: 'Have dinner without phones', done: true }, { n: 3, text: 'Pray together', done: false },
      { n: 4, text: 'Talk about one future goal', done: false }, { n: 5, text: 'Share a favourite memory', done: false }, { n: 6, text: 'Take a walk together', done: false }, { n: 7, text: 'Plan something to look forward to', done: false },
    ] },
    journal: [
      { id: 'j1', authorId: PARTNER, date: thisSunday, tag: 'Gratitude', text: 'Grateful for a slow Sunday. We didn’t do much and it was exactly right.' },
      { id: 'j2', authorId: ME, date: iso(addDays(TODAY, -5)), tag: 'Reflection', text: 'We handled the money conversation better than last time. We listened first.' },
      { id: 'j3', authorId: PARTNER, date: '2026-09-21', tag: 'Memory', text: 'Movie night. You fell asleep before the good part, again.' },
    ],
    appreciations: [
      { id: 'a1', fromId: PARTNER, date: thisSunday, text: 'I appreciate how you always remember the small things.' },
      { id: 'a2', fromId: ME, date: '2026-09-22', text: 'I appreciate you praying for my interview before I even asked.' },
    ],
    memories: [
      { id: 'm1', title: 'Our first Amorae date night', date: '2026-09-21', location: 'Lekki', note: 'We stayed until they stacked the chairs.', hasPhoto: true },
      { id: 'm2', title: 'Started our 30-day prayer journey', date: '2026-09-28', hasPhoto: true },
    ],
    milestones: [
      { id: 'd1', title: 'Adeola’s birthday', date: '2026-11-14', sub: 'Reminder a week before', reminder: true },
      { id: 'd2', title: 'Our anniversary', date: '2027-02-12', sub: 'Three years together', reminder: true },
      { id: 'd3', title: 'Started our 30-day prayer journey', date: '2026-09-28' },
      { id: 'd4', title: 'The day we started', date: '2024-02-12' },
    ],
    prefs: { newWeek: true, prayerReminder: true, reminderTime: '21:00', eventReminders: true, importantDates: true, appreciation: true, journal: true, goals: false, challenges: false },
  };
}

let cache: DB | null = null;
function load(): DB {
  if (cache) return cache;
  if (typeof window === 'undefined') return seed();
  try {
    const raw = localStorage.getItem(KEY);
    cache = raw ? (JSON.parse(raw) as DB) : seed();
  } catch { cache = seed(); }
  return cache!;
}
function save(db: DB) {
  cache = db;
  try { localStorage.setItem(KEY, JSON.stringify(db)); } catch { /* private mode: keep in memory */ }
}
export function resetDb() { cache = null; try { localStorage.removeItem(KEY); localStorage.removeItem(QUEUE_KEY); } catch { /* noop */ } }

/* ---------- offline queue ---------- */

export type QueuedOp = { id: string; op: string; payload: unknown; at: string };
export function readQueue(): QueuedOp[] { try { return JSON.parse(localStorage.getItem(QUEUE_KEY) ?? '[]'); } catch { return []; } }
function writeQueue(q: QueuedOp[]) { try { localStorage.setItem(QUEUE_KEY, JSON.stringify(q)); } catch { /* noop */ } }
export function enqueue(op: string, payload: unknown) { writeQueue([...readQueue(), { id: uid(), op, payload, at: new Date().toISOString() }]); }
/** Called when the network returns. With a real API each op would be POSTed; here local state is already applied. */
export async function flushQueue(): Promise<number> { const n = readQueue().length; await sleep(600); writeQueue([]); return n; }
export const online = () => (typeof navigator === 'undefined' ? true : navigator.onLine);

/* ---------- API surface ---------- */

const latency = () => sleep(online() ? 120 : 0);

export const api = {
  // auth
  async me() { await latency(); const db = load(); return { session: db.session, onboarded: db.onboarded, user: db.couple.me }; },
  async login() { const db = load(); db.session = true; db.onboarded = { couple: true, install: true, notifications: true }; save(db); },
  async register(name: string, email: string) { const db = seed(); db.session = true; db.couple.me = { ...db.couple.me, name, email }; db.couple.partner = null; save(db); },
  async logout() { const db = load(); db.session = false; save(db); },
  async setOnboarded(patch: Partial<DB['onboarded']>) { const db = load(); db.onboarded = { ...db.onboarded, ...patch }; save(db); },
  async deleteAccount() { resetDb(); },

  // couple
  async couple() { await latency(); return load().couple; },
  async createCouple() { const db = load(); db.onboarded.couple = true; save(db); return db.couple; },
  async joinCouple(code: string) { await sleep(400); const db = load(); if (code.replace(/\s/g, '').toUpperCase() !== db.couple.inviteCode) throw new Error('That code didn’t match. Check it with your partner and try again.'); db.onboarded.couple = true; db.couple.partner = { id: PARTNER, name: 'Adeola' }; save(db); return db.couple; },
  async updateProfile(patch: { name?: string; email?: string; timezone?: string }) { const db = load(); db.couple.me = { ...db.couple.me, ...patch }; save(db); return db.couple.me; },

  // prayer weeks
  async currentWeek() { await latency(); return load().weeks[0]; },
  async history() { await latency(); return load().weeks.slice(1); },
  async week(id: string) { await latency(); const w = load().weeks.find((x) => x.id === id); if (!w) throw new Error('Week not found'); return w; },
  async setCompleted(pointId: string, done: boolean) {
    const db = load(); const w = db.weeks[0];
    w.myCompleted = done ? Array.from(new Set([...w.myCompleted, pointId])) : w.myCompleted.filter((x) => x !== pointId);
    save(db); if (!online()) enqueue('complete', { pointId, done }); return w;
  },
  async savePoints(points: PrayerPoint[]) { const db = load(); db.weeks[0].points = points.map((p, i) => ({ ...p, position: i })); save(db); return db.weeks[0]; },
  async publish() { await sleep(500); const db = load(); db.weeks[0].status = 'published'; save(db); return db.weeks[0]; },
  async saveReflection(weekId: string, reflection: string) { const db = load(); const w = db.weeks.find((x) => x.id === weekId); if (w) w.reflection = reflection; save(db); },
  /** Demo helper: switch the current week into "your week" (draft) or "waiting" state. */
  async setDemoWeek(mode: 'partner' | 'mine' | 'waiting') {
    const db = load(); const w = db.weeks[0];
    if (mode === 'mine') { w.setterId = ME; w.status = 'draft'; w.points = pts([['Career direction', 'Clarity for our next season'], ['Family', 'Peace and health in both our homes'], ['Rest', 'That we’d slow down without guilt']]); w.myCompleted = []; w.partnerCompleted = []; }
    else if (mode === 'waiting') { w.setterId = PARTNER; w.status = 'waiting'; w.points = []; w.myCompleted = []; w.partnerCompleted = []; }
    else { const s = seed(); db.weeks[0] = s.weeks[0]; }
    save(db); return db.weeks[0];
  },

  // together
  async events() { await latency(); return load().events; },
  async event(id: string) { await latency(); const e = load().events.find((x) => x.id === id); if (!e) throw new Error('Event not found'); return e; },
  async saveEvent(e: Partial<Event> & { title: string; date: string }) {
    const db = load();
    if (e.id) { db.events = db.events.map((x) => (x.id === e.id ? { ...x, ...e } as Event : x)); }
    else { db.events.unshift({ checklist: [], done: false, ...e, id: uid() } as Event); }
    save(db); if (!online()) enqueue('event', e); return db.events;
  },
  async toggleChecklist(eventId: string, itemId: string, done: boolean) { const db = load(); const e = db.events.find((x) => x.id === eventId); if (e) e.checklist = e.checklist.map((c) => (c.id === itemId ? { ...c, done } : c)); save(db); return e; },
  async completeEvent(id: string, done: boolean) { const db = load(); const e = db.events.find((x) => x.id === id); if (e) e.done = done; save(db); return e; },

  async goals() { await latency(); return load().goals; },
  async goal(id: string) { await latency(); const g = load().goals.find((x) => x.id === id); if (!g) throw new Error('Goal not found'); return g; },
  async saveGoal(g: Partial<Goal> & { title: string }) {
    const db = load();
    if (g.id) db.goals = db.goals.map((x) => (x.id === g.id ? { ...x, ...g } as Goal : x));
    else db.goals.unshift({ target: 1, unit: 'count', progress: [], start: todayIso, end: todayIso, done: false, ...g, id: uid() } as Goal);
    save(db); return db.goals;
  },
  async addProgress(goalId: string, amount: number) { const db = load(); const g = db.goals.find((x) => x.id === goalId); if (g) g.progress.unshift({ id: uid(), by: ME, amount, date: todayIso }); save(db); if (!online()) enqueue('progress', { goalId, amount }); return g; },

  async challenge() { await latency(); return load().challenge; },
  async setChallengeDay(n: number, patch: { done?: boolean; skipped?: boolean }) { const db = load(); db.challenge.days = db.challenge.days.map((d) => (d.n === n ? { ...d, ...patch } : d)); save(db); return db.challenge; },

  async journal() { await latency(); return load().journal; },
  async addJournal(tag: JournalEntry['tag'], text: string) { const db = load(); db.journal.unshift({ id: uid(), authorId: ME, date: todayIso, tag, text }); save(db); return db.journal; },
  async appreciations() { await latency(); return load().appreciations; },
  async sendAppreciation(text: string) { await sleep(300); const db = load(); const a = { id: uid(), fromId: ME, date: todayIso, text }; db.appreciations.unshift(a); save(db); return a; },
  async undoAppreciation(id: string) { const db = load(); db.appreciations = db.appreciations.filter((a) => a.id !== id); save(db); },
  async memories() { await latency(); return load().memories; },
  async addMemory(m: Omit<Memory, 'id'>) { const db = load(); db.memories.unshift({ ...m, id: uid() }); save(db); return db.memories; },
  async milestones() { await latency(); return load().milestones; },
  async addMilestone(m: Omit<Milestone, 'id'>) { const db = load(); db.milestones.push({ ...m, id: uid() }); save(db); return db.milestones; },

  async prefs() { await latency(); return load().prefs; },
  async savePrefs(patch: Partial<NotificationPrefs>) { const db = load(); db.prefs = { ...db.prefs, ...patch }; save(db); return db.prefs; },
};
