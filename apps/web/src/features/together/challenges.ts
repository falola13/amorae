import type {
  Challenge,
  ChallengeCategory,
  ChallengeDay,
  ChallengeStatus,
  ChallengeTemplate,
} from "@/lib/api/types";
import {
  addDays,
  dayName,
  dayNum,
  daysUntil,
  iso,
  parse,
  shortMonth,
  weekdayDate,
} from "@/lib/dates";

export type Season = "" | "advent" | "lent";

/** Western Easter Sunday (Meeus/Jones/Butcher), as a local date. */
export function easterSunday(year: number): Date {
  const a = year % 19;
  const b = Math.floor(year / 100);
  const c = year % 100;
  const d = Math.floor(b / 4);
  const e = b % 4;
  const f = Math.floor((b + 8) / 25);
  const g = Math.floor((b - f + 1) / 3);
  const h = (19 * a + b - d - g + 15) % 30;
  const i = Math.floor(c / 4);
  const k = c % 4;
  const l = (32 + 2 * e + 2 * i - h - k) % 7;
  const m = Math.floor((a + 11 * h + 22 * l) / 451);
  const month = Math.floor((h + l - 7 * m + 114) / 31);
  const day = ((h + l - 7 * m + 114) % 31) + 1;
  return new Date(year, month - 1, day);
}

/** Advent is Dec 1-24; Lent runs from Ash Wednesday to Holy Saturday. */
export function currentSeason(now: Date): Season {
  if (now.getMonth() === 11 && now.getDate() <= 24) return "advent";
  const easter = easterSunday(now.getFullYear());
  const today = iso(now);
  if (today >= iso(addDays(easter, -46)) && today < iso(easter)) return "lent";
  return "";
}

/** "Opens tomorrow", "Opens Friday", or with the date once it is a week or more away. */
export function opensLabel(dateIso: string, todayIso: string): string {
  const n = daysUntil(dateIso, parse(todayIso));
  if (n <= 1) return "Opens tomorrow";
  if (n < 7) return `Opens ${dayName(dateIso)}`;
  return `Opens ${weekdayDate(dateIso)}`;
}

/** "Mon 12 Oct". */
export const dayDateLabel = (dateIso: string): string =>
  `${dayName(dateIso).slice(0, 3)} ${dayNum(dateIso)} ${shortMonth(dateIso)}`;

/** An earlier open day you have not marked either way. Still yours to mark. */
export const isCatchUp = (d: ChallengeDay, todayN: number): boolean =>
  d.open && d.n < todayN && !d.done && !d.skipped;

/** The number to show in "Day N of M", never past the last day. */
export const shownDay = (c: Pick<Challenge, "today_n" | "days">): number =>
  Math.max(1, Math.min(c.today_n, c.days.length));

/** Today's day when it is open and you have not marked it: what the home card offers. */
export function homeChallengeDay(c: Challenge | undefined): ChallengeDay | null {
  if (!c || c.status !== "active") return null;
  const d = c.days.find((x) => x.n === c.today_n);
  if (!d || !d.open || d.done || d.skipped) return null;
  return d;
}

/** How many days each of you marked done. */
export const doneCount = (c: Challenge): { mine: number; partner: number } => ({
  mine: c.days.filter((d) => d.done).length,
  partner: c.days.filter((d) => d.partner_done).length,
});

export const isOver = (status: ChallengeStatus): boolean => status !== "active";

export const overLabel = (status: ChallengeStatus): string =>
  status === "ended" ? "Ended early" : "Finished together";

export const CATEGORY_LABEL: Record<ChallengeCategory, string> = {
  connection: "Connection",
  faith: "Faith",
  service: "Service",
  season: "For the season",
};

const ORDER: ChallengeCategory[] = ["connection", "faith", "service"];

export interface TemplateGroup {
  category: ChallengeCategory;
  label: string;
  templates: ChallengeTemplate[];
}

/** Categories in a steady order. The seasonal group comes first while its season is
 *  on, this season's own before the other; out of season it comes last. Every
 *  template stays browsable either way. */
export function groupTemplates(templates: ChallengeTemplate[], season: Season): TemplateGroup[] {
  const cats: ChallengeCategory[] = season ? ["season", ...ORDER] : [...ORDER, "season"];
  return cats
    .map((category) => {
      let list = templates.filter((t) => t.category === category);
      if (category === "season" && season) {
        list = [
          ...list.filter((t) => t.season === season),
          ...list.filter((t) => t.season !== season),
        ];
      }
      return { category, label: CATEGORY_LABEL[category], templates: list };
    })
    .filter((g) => g.templates.length > 0);
}

/** Three to suggest next: this season's own first, then ones you have not done, then the
 *  rest. Templates of another season are left out of the suggestions. */
export function pickSuggestions(
  templates: ChallengeTemplate[],
  doneKeys: readonly string[],
  now: Date,
  count = 3,
): ChallengeTemplate[] {
  const season = currentSeason(now);
  const done = new Set(doneKeys);
  const eligible = templates.filter((t) => !t.season || t.season === season);
  const inSeason = eligible.filter((t) => t.season && t.season === season);
  const plain = eligible.filter((t) => !t.season);
  return [
    ...inSeason.filter((t) => !done.has(t.key)),
    ...plain.filter((t) => !done.has(t.key)),
    ...inSeason.filter((t) => done.has(t.key)),
    ...plain.filter((t) => done.has(t.key)),
  ].slice(0, count);
}

export const MIN_CUSTOM_DAYS = 3;
export const MAX_CUSTOM_DAYS = 40;
export const DEFAULT_CUSTOM_DAYS = 7;

/** Grows or shrinks the list of prompts to `n` days, keeping what is already written. */
export function resizePrompts(prompts: readonly string[], n: number): string[] {
  const size = Math.max(MIN_CUSTOM_DAYS, Math.min(MAX_CUSTOM_DAYS, Math.round(n) || 0));
  return Array.from({ length: size }, (_, i) => prompts[i] ?? "");
}

/** The first thing wrong with a custom challenge, before it is sent. */
export function customProblem(title: string, prompts: readonly string[]): string | null {
  if (!title.trim()) return "Give it a name.";
  if (prompts.length < MIN_CUSTOM_DAYS || prompts.length > MAX_CUSTOM_DAYS)
    return `Choose between ${MIN_CUSTOM_DAYS} and ${MAX_CUSTOM_DAYS} days.`;
  if (prompts.some((p) => !p.trim())) return "Each day needs a line.";
  return null;
}
