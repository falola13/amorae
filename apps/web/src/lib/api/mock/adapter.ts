import type { AxiosAdapter, AxiosResponse, InternalAxiosRequestConfig } from "axios";
import { AxiosError, AxiosHeaders } from "axios";

import type { Couple, Event, Goal, JournalEntry, Memory, Milestone, NotificationPrefs, PrayerPoint } from "../types";
import { ME, PARTNER, TODAY_ISO, load, pts, reset, save, seed, uid } from "./db";

// An axios adapter that answers the proposed v1 contract (docs/API.md) from
// the localStorage store. Same envelopes and status codes as the Go API, so
// the interceptors in lib/api/http.ts treat both identically.

type Handler = (p: Record<string, string>, body: Record<string, unknown>, query: URLSearchParams) => { status?: number; data?: unknown } | void;

const routes: [string, RegExp, Handler][] = [];
const on = (method: string, pattern: string, h: Handler) => {
  const keys: string[] = [];
  const re = new RegExp("^" + pattern.replace(/:(\w+)/g, (_, k) => { keys.push(k); return "([^/]+)"; }) + "$");
  routes.push([method, re, (p, b, q) => h(p, b, q)]);
  (re as RegExp & { keys: string[] }).keys = keys;
};

class Fail extends Error {
  constructor(public status: number, public code: string, message: string, public fields?: Record<string, string>) { super(message); }
}
const notFound = (what: string) => new Fail(404, `${what}_not_found`, `That ${what} isn’t here anymore.`);
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/* ---------- users and couple ---------- */
on("GET", "/users/me", () => ({ data: load().couple.me }));
// Mirrors the Go API: only display_name and timezone are patchable, and any
// other field (email included) is rejected, like its DisallowUnknownFields.
on("PATCH", "/users/me", (_p, b) => {
  const unknown = Object.keys(b).filter((k) => k !== "display_name" && k !== "timezone");
  if (unknown.length) throw new Fail(400, "invalid_json", "Request body must be valid JSON.");
  const db = load(); db.couple.me = { ...db.couple.me, ...(b as object), updated_at: new Date().toISOString() }; save(db); return { data: db.couple.me };
});
// Mock sign-in accepts any password, so here any non-empty current password counts.
on("PUT", "/users/me/email", (_p, b) => {
  const email = String(b.email ?? "").trim().toLowerCase(); const fields: Record<string, string> = {};
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) fields.email = "Enter a valid email address.";
  if (!String(b.current_password ?? "")) fields.current_password = "Enter your current password.";
  if (Object.keys(fields).length) throw new Fail(400, "validation_failed", "Some fields are invalid.", fields);
  const db = load(); db.couple.me = { ...db.couple.me, email, updated_at: new Date().toISOString() }; save(db); return { data: db.couple.me };
});
on("DELETE", "/users/me", () => { reset(); return { status: 204 }; });

on("GET", "/couples/me", () => ({ data: load().couple }));
on("POST", "/couples", () => { const db = load(); db.couple.onboarding.couple = true; db.couple.partner = null; save(db); return { status: 201, data: db.couple }; });
on("POST", "/couples/join", (_p, b) => {
  const norm = (c: string) => c.replace(/[^a-z0-9]/gi, "").toUpperCase();
  const db = load();
  if (norm(String(b.code ?? "")) !== norm(db.couple.invite_code)) throw new Fail(400, "validation_failed", "That code didn’t match.", { code: "Check it with your partner and try again." });
  db.couple.onboarding.couple = true; db.couple.partner = { id: PARTNER, display_name: "Adeola", role: "partner" }; save(db); return { data: db.couple };
});
// Mirrors the Go handler: couple is accepted but never stored, since being in
// a couple is that step's completed state.
on("PATCH", "/couples/me/onboarding", (_p, b) => {
  const db = load(); const { couple: _couple, ...steps } = b as Partial<Couple["onboarding"]>;
  db.couple.onboarding = { ...db.couple.onboarding, ...steps }; save(db); return { data: db.couple };
});

/* ---------- prayers ---------- */
on("GET", "/prayers/current", () => ({ data: load().weeks[0] }));
on("GET", "/prayers/history", () => ({ data: load().weeks.slice(1) }));
on("GET", "/prayers/weeks/:id", (p) => { const w = load().weeks.find((x) => x.id === p.id); if (!w) throw notFound("week"); return { data: w }; });
on("PUT", "/prayers/current/points", (_p, b) => {
  const db = load(); const points = (b.points as PrayerPoint[]).map((x, i) => ({ ...x, id: x.id || uid(), position: i }));
  if (points.length > 10) throw new Fail(400, "validation_failed", "Up to 10 prayers a week.", { points: "Up to 10 prayers a week." });
  db.weeks[0].points = points; save(db); return { data: db.weeks[0] };
});
on("POST", "/prayers/current/publish", () => { const db = load(); db.weeks[0].status = "published"; save(db); return { data: db.weeks[0] }; });
on("POST", "/prayers/points/:id/complete", (p) => { const db = load(); const w = db.weeks[0]; w.my_completed = Array.from(new Set([...w.my_completed, p.id])); save(db); return { data: w }; });
on("DELETE", "/prayers/points/:id/complete", (p) => { const db = load(); const w = db.weeks[0]; w.my_completed = w.my_completed.filter((x) => x !== p.id); save(db); return { data: w }; });
on("PATCH", "/prayers/weeks/:id/reflection", (p, b) => { const db = load(); const w = db.weeks.find((x) => x.id === p.id); if (!w) throw notFound("week"); w.reflection = String(b.reflection ?? ""); save(db); return { data: w }; });
// Demo only: the Go scheduler owns this for real. Lets the UI show all three Home states.
on("POST", "/prayers/current/demo", (_p, b) => {
  const db = load(); const w = db.weeks[0]; const mode = b.mode;
  if (mode === "mine") { w.setter_id = ME; w.status = "draft"; w.points = pts([["Career direction", "Clarity for our next season"], ["Family", "Peace and health in both our homes"], ["Rest", "That we’d slow down without guilt"]]); w.my_completed = []; w.partner_completed = []; }
  else if (mode === "waiting") { w.setter_id = PARTNER; w.status = "waiting"; w.points = []; w.my_completed = []; w.partner_completed = []; }
  else { db.weeks[0] = seed().weeks[0]; }
  save(db); return { data: db.weeks[0] };
});

/* ---------- events ---------- */
on("GET", "/events", () => ({ data: load().events }));
on("GET", "/events/:id", (p) => { const e = load().events.find((x) => x.id === p.id); if (!e) throw notFound("event"); return { data: e }; });
on("POST", "/events", (_p, b) => { const db = load(); const e = { checklist: [], done: false, ...(b as object), id: uid() } as unknown as Event; db.events.unshift(e); save(db); return { status: 201, data: e }; });
on("PATCH", "/events/:id", (p, b) => { const db = load(); const e = db.events.find((x) => x.id === p.id); if (!e) throw notFound("event"); Object.assign(e, b); save(db); return { data: e }; });
on("POST", "/events/:id/complete", (p) => { const db = load(); const e = db.events.find((x) => x.id === p.id); if (!e) throw notFound("event"); e.done = true; save(db); return { data: e }; });
on("DELETE", "/events/:id/complete", (p) => { const db = load(); const e = db.events.find((x) => x.id === p.id); if (!e) throw notFound("event"); e.done = false; save(db); return { data: e }; });
on("PATCH", "/events/:id/checklist/:item", (p, b) => { const db = load(); const e = db.events.find((x) => x.id === p.id); if (!e) throw notFound("event"); e.checklist = e.checklist.map((c) => (c.id === p.item ? { ...c, done: Boolean(b.done) } : c)); save(db); return { data: e }; });

/* ---------- goals ---------- */
on("GET", "/goals", () => ({ data: load().goals }));
on("GET", "/goals/:id", (p) => { const g = load().goals.find((x) => x.id === p.id); if (!g) throw notFound("goal"); return { data: g }; });
on("POST", "/goals", (_p, b) => { const db = load(); const g = { target: 1, unit: "count", progress: [], start_date: TODAY_ISO, end_date: TODAY_ISO, done: false, ...(b as object), id: uid() } as unknown as Goal; db.goals.unshift(g); save(db); return { status: 201, data: g }; });
on("POST", "/goals/:id/progress", (p, b) => { const db = load(); const g = db.goals.find((x) => x.id === p.id); if (!g) throw notFound("goal"); g.progress.unshift({ id: uid(), user_id: ME, amount: Number(b.amount), date: TODAY_ISO }); save(db); return { data: g }; });

/* ---------- challenges, journal, appreciation, memories, milestones ---------- */
on("GET", "/challenges/current", () => ({ data: load().challenge }));
on("PATCH", "/challenges/current/days/:n", (p, b) => { const db = load(); db.challenge.days = db.challenge.days.map((d) => (d.n === Number(p.n) ? { ...d, ...(b as object) } : d)); save(db); return { data: db.challenge }; });

on("GET", "/journal", () => ({ data: load().journal }));
on("POST", "/journal", (_p, b) => { const db = load(); const j: JournalEntry = { id: uid(), author_id: ME, date: TODAY_ISO, tag: b.tag as JournalEntry["tag"], text: String(b.text) }; db.journal.unshift(j); save(db); return { status: 201, data: j }; });

on("GET", "/appreciations", () => ({ data: load().appreciations }));
on("POST", "/appreciations", (_p, b) => { const db = load(); const a = { id: uid(), from_id: ME, date: TODAY_ISO, text: String(b.text) }; db.appreciations.unshift(a); save(db); return { status: 201, data: a }; });
on("DELETE", "/appreciations/:id", (p) => { const db = load(); db.appreciations = db.appreciations.filter((a) => a.id !== p.id); save(db); return { status: 204 }; });

on("GET", "/memories", () => ({ data: load().memories }));
on("POST", "/memories", (_p, b) => { const db = load(); const m = { has_photo: false, ...(b as object), id: uid() } as Memory; db.memories.unshift(m); save(db); return { status: 201, data: m }; });

on("GET", "/milestones", () => ({ data: load().milestones }));
on("POST", "/milestones", (_p, b) => { const db = load(); const m = { ...(b as object), id: uid() } as Milestone; db.milestones.push(m); save(db); return { status: 201, data: m }; });

on("GET", "/notifications/preferences", () => ({ data: load().prefs }));
on("PATCH", "/notifications/preferences", (_p, b) => { const db = load(); db.prefs = { ...db.prefs, ...(b as Partial<NotificationPrefs>) }; save(db); return { data: db.prefs }; });
on("POST", "/notifications/subscribe", () => ({ status: 204 }));

/* ---------- the adapter ---------- */
export const mockAdapter: AxiosAdapter = async (config: InternalAxiosRequestConfig): Promise<AxiosResponse> => {
  const method = (config.method ?? "get").toUpperCase();
  const url = new URL(config.url ?? "/", "http://mock.local");
  const path = url.pathname.replace(/^\/api\/v1/, "").replace(/\/$/, "") || "/";
  const body = typeof config.data === "string" && config.data ? (JSON.parse(config.data) as Record<string, unknown>) : ((config.data as Record<string, unknown>) ?? {});
  await sleep(navigator.onLine ? 90 : 0);

  const respond = (status: number, data: unknown): AxiosResponse => ({ status, statusText: "", headers: {}, config, data });
  const reject = (status: number, code: string, message: string, fields?: Record<string, string>) => {
    const res = respond(status, { error: { code, message, fields } });
    return Promise.reject(new AxiosError(message, String(status), config, undefined, res));
  };

  for (const [m, re, handler] of routes) {
    if (m !== method) continue;
    const match = path.match(re);
    if (!match) continue;
    const keys = (re as RegExp & { keys: string[] }).keys ?? [];
    const params = Object.fromEntries(keys.map((k, i) => [k, decodeURIComponent(match[i + 1])]));
    try {
      const out = handler(params, body, url.searchParams) ?? {};
      const status = out.status ?? 200;
      return respond(status, status === 204 ? "" : { data: out.data });
    } catch (e) {
      if (e instanceof Fail) return reject(e.status, e.code, e.message, e.fields);
      return reject(500, "internal_error", "Something went wrong. Please try again.");
    }
  }
  config.headers = config.headers ?? new AxiosHeaders();
  return reject(404, "route_not_found", `No mock for ${method} ${path}`);
};
