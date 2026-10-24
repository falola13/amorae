"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, type ReactNode } from "react";

import { Icon } from "@/components/icons";
import { cx } from "@/components/ui/kit";
import { useSaveEvent } from "@/features/together/hooks";
import type { Anniversary, AnniversaryLetter, Appreciation, Couple, Memory } from "@/lib/api/types";
import { longDateYear, parse, shortMonth, time12, weekdayDate } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { IDEAS, QUESTIONS, anniversaryPrayer, symbolFor } from "./content";
import { count, daysTogether, notesOfTheYear, photosOfTheYear, yearTiles } from "./derive";

/* ---------- Pieces ---------- */

/** Two rings, linked: the mark's two arcs grown into a keepsake. */
export function Rings({ size = 76 }: { size?: number }) {
  return (
    <svg width={size} height={size * 0.62} viewBox="0 0 100 62" aria-hidden="true">
      <defs>
        <linearGradient id="ring-gold" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#fff1c9" />
          <stop offset="0.45" stopColor="#d9ad5f" />
          <stop offset="1" stopColor="#8a5d1f" />
        </linearGradient>
      </defs>
      <circle cx="37" cy="31" r="24" fill="none" stroke="url(#ring-gold)" strokeWidth="4" />
      <circle cx="63" cy="31" r="24" fill="none" stroke="url(#ring-gold)" strokeWidth="4" />
      {/* Redraw the left ring's crossing on top, so they link rather than overlap. */}
      <path
        d="M58.2 14.6 A24 24 0 0 1 58.2 47.4"
        fill="none"
        stroke="url(#ring-gold)"
        strokeWidth="4"
      />
    </svg>
  );
}

/** A section's title: a gold flourish, then the words in the occasion's serif. */
export function Heading({ children, sub }: { children: ReactNode; sub?: ReactNode }) {
  return (
    <div className="mb-5 flex flex-col items-center text-center">
      <svg width="64" height="12" viewBox="0 0 64 12" aria-hidden="true" className="text-gold">
        <path d="M2 6 H24 M40 6 H62" stroke="currentColor" strokeWidth="1" strokeLinecap="round" />
        <path d="M32 1.5 L36 6 L32 10.5 L28 6 Z" fill="currentColor" />
      </svg>
      <h2 className="m-0 mt-2 font-serif text-[31px] font-medium italic leading-[1.1] tracking-[-0.01em] text-plum-dark">
        {children}
      </h2>
      {sub ? <p className="m-0 mt-1.5 text-support text-stone">{sub}</p> : null}
    </div>
  );
}

const pad = (n: number) => String(n).padStart(2, "0");

/** Re-renders every second while mounted. */
function useNow() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(t);
  }, []);
  return now;
}

function Ticker({ cells }: { cells: [string, string][] }) {
  return (
    <div className="flex justify-center gap-2" role="timer">
      {cells.map(([value, label]) => (
        <div
          key={label}
          className="flex min-w-[58px] flex-col items-center rounded-[14px] border border-white/15 bg-white/[0.06] px-2 pb-1.5 pt-2"
        >
          <span className="lining font-serif text-[26px] font-semibold leading-none text-[#fff3dc]">
            {value}
          </span>
          <span className="mt-1 text-[10px] font-bold uppercase tracking-[0.14em] text-[#e9c886]/80">
            {label}
          </span>
        </div>
      ))}
    </div>
  );
}

/** On the day: how long, to the second, since they started. */
function TogetherFor({ started, years }: { started: string; years: number }) {
  const now = useNow();
  const lastAnniversary = new Date(parse(started));
  lastAnniversary.setFullYear(lastAnniversary.getFullYear() + years);
  const ms = Math.max(0, now.getTime() - lastAnniversary.getTime());
  const days = Math.floor(ms / 86400000);
  const rest = ms % 86400000;
  return (
    <Ticker
      cells={[
        [String(years), years === 1 ? "year" : "years"],
        [String(days), days === 1 ? "day" : "days"],
        [pad(Math.floor(rest / 3600000)), "hrs"],
        [pad(Math.floor((rest % 3600000) / 60000)), "min"],
        [pad(Math.floor((rest % 60000) / 1000)), "sec"],
      ]}
    />
  );
}

/** Before the day: how long until it. */
function Countdown({ on }: { on: string }) {
  const now = useNow();
  const ms = Math.max(0, parse(on).getTime() - now.getTime());
  const days = Math.floor(ms / 86400000);
  const rest = ms % 86400000;
  const cells: [string, string][] = [
    [pad(Math.floor(rest / 3600000)), "hrs"],
    [pad(Math.floor((rest % 3600000) / 60000)), "min"],
    [pad(Math.floor((rest % 60000) / 1000)), "sec"],
  ];
  if (days > 0) cells.unshift([String(days), days === 1 ? "day" : "days"]);
  return <Ticker cells={cells} />;
}

// Fixed, so the server and every render agree.
const STARS = [
  [8, 14, 0],
  [18, 62, 1.2],
  [27, 30, 2.1],
  [12, 84, 0.6],
  [86, 18, 1.6],
  [92, 48, 0.3],
  [78, 76, 2.4],
  [70, 10, 0.9],
  [40, 8, 1.9],
  [60, 90, 1.1],
  [95, 88, 2.8],
  [4, 46, 1.4],
] as const;

/* ---------- Hero ---------- */

/**
 * The keepsake at the top of the page. On the day: a great gold numeral,
 * the time together ticking, and the glass to raise. Before it: the same
 * evening, counting down.
 */
export function Keepsake({
  a,
  couple,
  showDay,
  onCheers,
  cheersBusy,
}: {
  a: Anniversary;
  couple: Couple;
  showDay: boolean;
  onCheers: () => void;
  cheersBusy: boolean;
}) {
  const partner = couple.partner?.display_name.split(" ")[0];
  const me = couple.me.display_name.split(" ")[0];
  const started = couple.started_on;
  const date = parse(a.on);
  return (
    <section className="relative -mx-2 mt-1 overflow-hidden rounded-[30px] bg-keepsake px-6 pb-8 pt-9 text-center text-[#fbeef2] shadow-[0_30px_60px_-30px_rgba(61,20,40,0.65)]">
      <div aria-hidden="true" className="pointer-events-none absolute inset-0">
        {STARS.map(([x, y, d]) => (
          <span
            key={`${x}-${y}`}
            className="absolute h-1 w-1 animate-twinkle rounded-full bg-[#fff1c9] shadow-[0_0_8px_rgba(255,230,180,0.9)]"
            style={{ left: `${x}%`, top: `${y}%`, animationDelay: `${d}s` }}
          />
        ))}
      </div>

      <div className="relative flex flex-col items-center">
        <Rings />
        <div className="mt-4 text-[11px] font-bold uppercase tracking-[0.28em] text-[#e9c886]">
          {weekdayDate(a.on).split(" ")[0]} · {date.getDate()} {shortMonth(a.on)} ·{" "}
          {date.getFullYear()}
        </div>

        <div className="lining mt-2 font-serif text-[148px] font-semibold leading-[0.82] tracking-[-0.04em]">
          <span className="animate-shimmer text-foil">{a.years}</span>
        </div>
        <div className="mt-1 font-serif text-[34px] font-medium italic leading-tight text-[#fbe4ec]">
          {showDay
            ? `${a.years === 1 ? "year" : "years"} together`
            : a.days_until === 1
              ? "years, tomorrow"
              : `years, in ${a.days_until} days`}
        </div>

        <div className="mt-4 flex items-center gap-3 text-[12px] font-bold uppercase tracking-[0.3em] text-[#f0cbd8]">
          <span className="h-px w-8 bg-linear-to-r from-transparent to-[#e9c886]" />
          {me}
          {partner ? (
            <span className="font-serif text-[18px] normal-case italic tracking-normal text-[#e9c886]">
              &amp;
            </span>
          ) : null}
          {partner ?? ""}
          <span className="h-px w-8 bg-linear-to-l from-transparent to-[#e9c886]" />
        </div>

        <div className="mt-7 w-full">
          {showDay && started ? <TogetherFor started={started} years={a.years} /> : null}
          {!showDay ? <Countdown on={a.on} /> : null}
        </div>
        {showDay && started ? (
          <p className="m-0 mt-3 text-[13px] text-[#dcb6c5]">
            since {longDateYear(started)} · {count(daysTogether(started, a.on))} days of choosing
            each other
          </p>
        ) : null}

        {showDay ? (
          <button
            type="button"
            onClick={onCheers}
            disabled={cheersBusy}
            className="press mt-7 inline-flex h-[52px] items-center gap-2.5 rounded-full px-7 text-[16px] font-bold text-[#3d1428] shadow-[0_10px_30px_-10px_rgba(233,199,127,0.8)]"
            style={{ background: "linear-gradient(100deg, #f6e2ae, #e2b866 55%, #f3d79b)" }}
          >
            <GlassIcon />
            Raise a glass
          </button>
        ) : a.mine ? null : a.writable ? (
          <Link
            href={routes.ourDayLetter}
            className="press mt-7 inline-flex h-[52px] items-center gap-2.5 rounded-full bg-[#fbeef2] px-7 text-[16px] font-bold text-[#4a1a35] no-underline"
          >
            <Icon name="pencil" size={18} />
            Write {partner ?? "them"} a letter
          </Link>
        ) : null}
      </div>
    </section>
  );
}

function GlassIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 24 24"
      aria-hidden="true"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.7"
      strokeLinecap="round"
    >
      <path d="M6 3h5l-.5 6a2 2 0 0 1-4 0zM8.5 11v8M6.5 20h4" />
      <path d="M13 3h5l-.5 6a2 2 0 0 1-4 0zM15.5 11v8M13.5 20h4" />
    </svg>
  );
}

/* ---------- Letters ---------- */

/** The envelope: wine paper, a gold wax seal. Tapped, the flap lifts and the letter rises out. */
function SealedEnvelope({
  from,
  note,
  onOpen,
}: {
  from: string;
  note: string;
  onOpen?: () => void;
}) {
  const [lifting, setLifting] = useState(false);
  const Tag = onOpen ? "button" : "div";
  return (
    <Tag
      type={onOpen ? "button" : undefined}
      onClick={
        onOpen
          ? () => {
              setLifting(true);
              setTimeout(onOpen, 560);
            }
          : undefined
      }
      className={cx(
        "relative block aspect-[1.55] w-full overflow-hidden rounded-[18px] text-left [perspective:900px]",
        onOpen && "press",
      )}
      style={{ background: "linear-gradient(160deg, #6b2a4c, #4a1a35)" }}
    >
      {/* The flap, folding up from the top edge. */}
      <span
        aria-hidden="true"
        className={cx("absolute inset-x-0 top-0 h-[50%] origin-top", lifting && "animate-flap")}
        style={{
          background: "linear-gradient(180deg, #7d3358, #5c2343)",
          clipPath: "polygon(0 0, 100% 0, 50% 100%)",
          filter: "drop-shadow(0 4px 6px rgba(0,0,0,0.25))",
        }}
      />
      <span
        aria-hidden="true"
        className="absolute inset-x-0 bottom-0 h-[60%]"
        style={{
          background: "linear-gradient(0deg, #4a1a35, #5a2042)",
          clipPath: "polygon(0 100%, 100% 100%, 100% 0, 50% 45%, 0 0)",
        }}
      />
      {/* The seal. */}
      <span
        aria-hidden="true"
        className="absolute left-1/2 top-[48%] flex h-14 w-14 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full"
        style={{
          background: "radial-gradient(circle at 35% 30%, #f6e2ae, #c99a4b 55%, #8a5d1f)",
          boxShadow: "0 4px 10px rgba(0,0,0,0.35), inset 0 0 0 3px rgba(255,241,201,0.35)",
        }}
      >
        <Icon name="heart" size={22} strokeWidth={2} className="text-[#5a2042]" />
      </span>
      <span className="absolute inset-x-0 bottom-4 flex flex-col items-center">
        <span className="font-serif text-[22px] italic text-[#fbe4ec]">from {from}</span>
        <span className="mt-0.5 text-[12px] font-bold uppercase tracking-[0.18em] text-[#e9c886]">
          {note}
        </span>
      </span>
    </Tag>
  );
}

function LetterPaper({
  letter,
  from,
  to,
}: {
  letter: AnniversaryLetter;
  from: string;
  to: string;
}) {
  return (
    <article
      data-selectable
      // Real paper in either theme: cream stock, ruled, ink and wine.
      className="animate-unfold rounded-[18px] px-6 pb-7 pt-7 shadow-[0_24px_50px_-24px_rgba(20,5,12,0.7)]"
      style={{
        backgroundColor: "#fbf5ec",
        backgroundImage:
          "repeating-linear-gradient(to bottom, transparent 0, transparent 31px, rgba(160,120,110,0.22) 31px, rgba(160,120,110,0.22) 32px)",
        backgroundPosition: "0 28px",
      }}
    >
      <p className="m-0 font-serif text-[26px] font-medium italic leading-[32px] text-[#5f1f3e]">
        Dear {to},
      </p>
      <div className="mt-2 whitespace-pre-wrap text-[17px] leading-[32px] text-[#2a1d22]">
        {letter.text}
      </div>
      <p className="m-0 mt-5 text-right font-serif text-[24px] italic leading-[32px] text-[#7b2c4f]">
        with love, {from}
      </p>
    </article>
  );
}

export function Letters({
  a,
  couple,
  showDay,
}: {
  a: Anniversary;
  couple: Couple;
  showDay: boolean;
}) {
  const [open, setOpen] = useState(false);
  const me = couple.me.display_name.split(" ")[0];
  const partner = couple.partner?.display_name.split(" ")[0] ?? "your partner";
  const when = a.days_until === 1 ? "tomorrow" : weekdayDate(a.on);
  return (
    <section className="mt-14">
      <Heading sub={showDay ? undefined : `Sealed until ${when}`}>Letters</Heading>
      <div className="flex flex-col gap-4">
        {a.theirs ? (
          open ? (
            <LetterPaper letter={a.theirs} from={partner} to={me} />
          ) : (
            <SealedEnvelope
              from={partner}
              note="Tap to break the seal"
              onOpen={() => setOpen(true)}
            />
          )
        ) : a.theirs_sealed ? (
          <SealedEnvelope from={partner} note={`Opens ${when}`} />
        ) : showDay ? (
          <p className="m-0 text-center text-support text-stone">
            If {partner} writes you one today, it will be here.
          </p>
        ) : null}

        {a.mine ? (
          <Link
            href={routes.ourDayLetter}
            className="press flex items-center gap-4 rounded-[18px] border border-line bg-surface px-5 py-4 text-ink no-underline"
          >
            <Icon name="letter" size={22} className="shrink-0 text-plum" />
            <span className="flex grow flex-col">
              <span className="text-[16px] font-semibold">Your letter to {partner}</span>
              <span className="text-support text-stone">
                {a.days_until === 0
                  ? `${partner} can read it now`
                  : `Sealed until ${when} · tap to change it`}
              </span>
            </span>
            <Icon name="right" size={18} className="text-stone" />
          </Link>
        ) : a.writable ? (
          <Link
            href={routes.ourDayLetter}
            className="press flex items-center gap-4 rounded-[18px] border border-dashed border-gold/60 px-5 py-4 text-ink no-underline"
          >
            <Icon name="pencil" size={22} className="shrink-0 text-plum" />
            <span className="flex grow flex-col">
              <span className="text-[16px] font-semibold">Write {partner} a letter</span>
              <span className="text-support text-stone">
                {a.days_until === 0
                  ? "They can read it as soon as you keep it."
                  : `Kept sealed until ${when}.`}
              </span>
            </span>
            <Icon name="right" size={18} className="text-stone" />
          </Link>
        ) : null}
      </div>
    </section>
  );
}

/* ---------- The year ---------- */

/** The year in numbers, set like a printed programme. */
export function YearInNumbers({ a }: { a: Anniversary }) {
  const tiles = yearTiles(a.year);
  if (tiles.length === 0) return null;
  const [lead, ...rest] = tiles;
  return (
    <section className="mt-16">
      <Heading
        sub={`${shortMonth(a.year.from)} ${a.year.from.slice(0, 4)} – ${shortMonth(a.year.to)} ${a.year.to.slice(0, 4)}`}
      >
        The year, in numbers
      </Heading>
      <div className="overflow-hidden rounded-[22px] border border-line bg-surface">
        <div className="flex flex-col items-center border-b border-line bg-occasion px-5 py-7 text-center">
          <span className="lining font-serif text-[76px] font-semibold leading-none tracking-[-0.03em]">
            <span className="text-foil">{count(lead.n)}</span>
          </span>
          <span className="mt-2 text-[12px] font-bold uppercase tracking-[0.2em] text-stone">
            {lead.label}
          </span>
        </div>
        <div className="grid grid-cols-2">
          {rest.map((t, i) => (
            <div
              key={t.label}
              className={cx(
                "flex flex-col items-center px-3 py-5 text-center",
                i % 2 === 0 && "border-r border-line",
                i < rest.length - (rest.length % 2 === 0 ? 2 : 1) && "border-b border-line",
                rest.length % 2 === 1 && i === rest.length - 1 && "col-span-2 border-r-0",
              )}
            >
              <span className="lining font-serif text-[40px] font-semibold leading-none text-plum-dark">
                {count(t.n)}
              </span>
              <span className="mt-1.5 text-[11px] font-bold uppercase tracking-[0.16em] text-stone">
                {t.label}
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

const TILTS = ["-rotate-3", "rotate-2", "-rotate-1", "rotate-3", "-rotate-2", "rotate-1"];

/** The year's photographs, as polaroids laid across the table. */
export function Moments({ a, memories }: { a: Anniversary; memories: Memory[] }) {
  const photos = photosOfTheYear(memories, a.year, 8);
  if (photos.length === 0) return null;
  return (
    <section className="mt-16">
      <Heading sub="From the year you just had">Moments you kept</Heading>
      <div className="-mx-6 flex snap-x snap-mandatory gap-5 overflow-x-auto px-8 pb-6 pt-2 [scrollbar-width:none] md:-mx-8">
        {photos.map((m, i) => (
          <Link
            key={m.id}
            href={routes.memories}
            className={cx(
              "press block w-[62%] max-w-[240px] shrink-0 snap-center bg-[#fffdf8] p-2.5 pb-3 text-ink no-underline shadow-[0_14px_30px_-14px_rgba(40,15,25,0.45)]",
              TILTS[i % TILTS.length],
            )}
          >
            {/* eslint-disable-next-line @next/next/no-img-element -- already optimised at the CDN */}
            <img
              src={m.photo_url}
              alt={m.title}
              className="aspect-square w-full bg-photo object-cover"
            />
            <span className="mt-2.5 block truncate text-center font-serif text-[19px] italic text-[#3d1428]">
              {m.title}
            </span>
            <span className="block text-center text-[11px] font-bold uppercase tracking-[0.14em] text-[#8f857b]">
              {shortMonth(m.date)} {m.date.slice(0, 4)}
            </span>
          </Link>
        ))}
      </div>
    </section>
  );
}

export function InYourWords({
  a,
  notes,
  nameOf,
}: {
  a: Anniversary;
  notes: Appreciation[];
  nameOf: (id: string) => string;
}) {
  const picked = notesOfTheYear(notes, a.year);
  if (picked.length === 0) return null;
  return (
    <section className="mt-14">
      <Heading sub="Notes you sent each other this year">In your words</Heading>
      <div className="flex flex-col gap-8">
        {picked.map((n) => (
          <figure key={n.id} className="m-0 flex flex-col items-center px-2 text-center">
            <span aria-hidden="true" className="font-serif text-[64px] leading-[0.5] text-gold">
              &ldquo;
            </span>
            <blockquote
              data-selectable
              className="m-0 mt-3 font-serif text-[25px] font-medium italic leading-[1.3] text-ink"
            >
              {n.text}
            </blockquote>
            <figcaption className="mt-3 text-[11px] font-bold uppercase tracking-[0.18em] text-stone">
              {nameOf(n.from_id)} · {shortMonth(n.date)} {n.date.slice(0, 4)}
            </figcaption>
          </figure>
        ))}
      </div>
    </section>
  );
}

/* ---------- More for the day ---------- */

const ORDINAL = ["", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"];

/** What this year is traditionally marked with, stitched like the leather of year three. */
export function YearSymbol({ years }: { years: number }) {
  const s = symbolFor(years);
  if (!s) return null;
  return (
    <section className="mt-16">
      <Heading>
        Year {ORDINAL[years] ?? years} is {s.material.toLowerCase()}
      </Heading>
      <div className="rounded-[22px] bg-occasion p-2">
        <div className="rounded-[16px] border-2 border-dashed border-gold/50 px-6 py-6 text-center">
          <p className="m-0 font-serif text-[22px] italic leading-[1.4] text-ink">{s.meaning}</p>
          <p className="m-0 mt-4 text-[11px] font-bold uppercase tracking-[0.18em] text-gold">
            An idea
          </p>
          <p className="m-0 mt-1 text-support text-stone">{s.idea}</p>
        </div>
      </div>
    </section>
  );
}

/** A small deck of questions for the evening, one at a time. */
export function Questions({ on }: { on: string }) {
  // The same order on both phones on the same day, a different one next year.
  const order = useMemo(() => {
    const seed = Number(on.replaceAll("-", ""));
    return QUESTIONS.map((q, i) => ({ q, k: Math.sin(seed + i) }))
      .sort((x, y) => x.k - y.k)
      .map((x) => x.q);
  }, [on]);
  const [i, setI] = useState(0);
  const next = () => setI((n) => (n + 1) % order.length);
  return (
    <section className="mt-16">
      <Heading sub="Over dinner, or the drive home">Questions for tonight</Heading>
      <div className="relative mx-2 h-[230px]">
        <div
          aria-hidden="true"
          className="absolute inset-0 rotate-3 rounded-[22px] border border-line bg-paper"
        />
        <div
          aria-hidden="true"
          className="absolute inset-0 -rotate-2 rounded-[22px] border border-line bg-surface"
        />
        <button
          key={i}
          type="button"
          onClick={next}
          className="press absolute inset-0 flex animate-rise flex-col items-center justify-center rounded-[22px] border border-line bg-surface px-7 text-center shadow-[0_18px_40px_-28px_rgba(60,20,40,0.5)]"
        >
          <span className="text-[11px] font-bold uppercase tracking-[0.2em] text-gold">
            {i + 1} of {order.length}
          </span>
          <span className="mt-3 font-serif text-[25px] font-medium italic leading-[1.3] text-ink">
            {order[i]}
          </span>
          <span className="mt-4 text-[13px] font-semibold text-plum">Tap for the next one</span>
        </button>
      </div>
    </section>
  );
}

/** Small plans for the day; one tap puts it in their plans for the day. */
export function Ideas({ on }: { on: string }) {
  const save = useSaveEvent();
  const [added, setAdded] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState<string | null>(null);
  return (
    <section className="mt-16">
      <Heading sub="Tap one to add it to your plans">Ideas for the day</Heading>
      <div className="flex flex-col overflow-hidden rounded-[22px] border border-line bg-surface">
        {IDEAS.map((idea, i) => {
          const id = added[idea.title];
          return (
            <div
              key={idea.title}
              className={cx("flex items-center gap-3 px-5 py-3.5", i > 0 && "border-t border-line")}
            >
              <span className="w-[54px] shrink-0 font-serif text-[17px] italic text-gold">
                {idea.time ? time12(idea.time).replace(":00", "") : "·"}
              </span>
              <span className="flex grow flex-col">
                <span className="text-[16px] font-semibold">{idea.title}</span>
                <span className="text-[13px] text-stone">{idea.detail}</span>
              </span>
              {id ? (
                <Link
                  href={routes.event(id)}
                  className="press flex h-9 shrink-0 items-center gap-1 rounded-full bg-green-tint px-3 text-[13px] font-bold text-green no-underline"
                >
                  <Icon name="check" size={16} /> Added
                </Link>
              ) : (
                <button
                  type="button"
                  disabled={busy !== null}
                  onClick={() => {
                    setBusy(idea.title);
                    save.mutate(
                      {
                        input: {
                          title: idea.title,
                          date: on,
                          ...(idea.time ? { start_time: idea.time } : {}),
                          notes: idea.detail,
                          kind: "together",
                        },
                      },
                      {
                        onSuccess: (e) => setAdded((m) => ({ ...m, [idea.title]: e.id })),
                        onSettled: () => setBusy(null),
                      },
                    );
                  }}
                  className="press flex h-9 shrink-0 items-center gap-1 rounded-full bg-plum-tint px-3 text-[13px] font-bold text-plum"
                >
                  <Icon name="plus" size={16} /> {busy === idea.title ? "…" : "Add"}
                </button>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}

const amenKey = (on: string) => `amorae.anniversary.amen.${on}`;

/** A prayer for the year ahead, in their names, to pray together. */
export function Prayer({ a, couple }: { a: Anniversary; couple: Couple }) {
  const me = couple.me.display_name.split(" ")[0];
  const partner = couple.partner?.display_name.split(" ")[0] ?? "us";
  const lines = anniversaryPrayer(a.years, me, partner);
  const [amen, setAmen] = useState(() => {
    try {
      return typeof window !== "undefined" && window.localStorage.getItem(amenKey(a.on)) === "1";
    } catch {
      return false;
    }
  });
  return (
    <section className="mt-16">
      <Heading sub="Pray it together, out loud">
        A prayer for year {ORDINAL[a.years + 1] ?? a.years + 1}
      </Heading>
      <div className="rounded-[22px] border border-line bg-surface px-6 py-7 text-center">
        <div className="flex flex-col gap-3.5">
          {lines.map((l) => (
            <p key={l} className="m-0 font-serif text-[21px] leading-[1.45] text-ink">
              {l}
            </p>
          ))}
        </div>
        <button
          type="button"
          onClick={() => {
            setAmen(true);
            try {
              window.localStorage.setItem(amenKey(a.on), "1");
            } catch {
              // Remembering it is a nicety; the prayer was prayed either way.
            }
          }}
          disabled={amen}
          className={cx(
            "press mt-6 inline-flex h-12 items-center justify-center rounded-full px-8 font-serif text-[24px] italic",
            amen ? "text-gold" : "bg-plum text-surface",
          )}
        >
          {amen ? "Amen." : "Amen"}
        </button>
      </div>
    </section>
  );
}

/** Where it began: the day they started, the first memory, the first note. */
export function WhereItBegan({
  started,
  memories,
  notes,
  nameOf,
}: {
  started?: string;
  memories: Memory[];
  notes: Appreciation[];
  nameOf: (id: string) => string;
}) {
  const firstMemory = [...memories].sort((x, y) => x.date.localeCompare(y.date))[0];
  const firstNote = [...notes].sort((x, y) => x.date.localeCompare(y.date))[0];
  const steps: { when: string; title: string; text?: string }[] = [];
  if (started) steps.push({ when: longDateYear(started), title: "The day it began" });
  if (firstMemory)
    steps.push({
      when: longDateYear(firstMemory.date),
      title: "Your first kept memory",
      text: firstMemory.title,
    });
  if (firstNote)
    steps.push({
      when: longDateYear(firstNote.date),
      title: `The first note, from ${nameOf(firstNote.from_id)}`,
      text: `“${firstNote.text}”`,
    });
  if (steps.length < 2) return null;
  return (
    <section className="mt-16">
      <Heading>Where it began</Heading>
      <ol className="m-0 flex list-none flex-col p-0">
        {steps.map((s, i) => (
          <li key={s.title} className="relative flex gap-4 pb-6 pl-1 last:pb-0">
            {i < steps.length - 1 ? (
              <span
                aria-hidden="true"
                className="absolute left-[9px] top-5 h-full w-px bg-gold/40"
              />
            ) : null}
            <span
              aria-hidden="true"
              className="relative mt-1.5 h-[11px] w-[11px] shrink-0 rounded-full border-2 border-gold bg-bg"
            />
            <span className="flex flex-col">
              <span className="text-[11px] font-bold uppercase tracking-[0.16em] text-gold">
                {s.when}
              </span>
              <span className="mt-0.5 text-[16px] font-semibold">{s.title}</span>
              {s.text ? (
                <span className="mt-0.5 font-serif text-[19px] italic text-stone">{s.text}</span>
              ) : null}
            </span>
          </li>
        ))}
      </ol>
    </section>
  );
}

/** Letters from anniversaries gone by. */
export function PastLetters({
  letters,
  nameOf,
}: {
  letters: AnniversaryLetter[];
  nameOf: (id: string) => string;
}) {
  if (letters.length === 0) return null;
  return (
    <section className="mt-16">
      <Heading>From other years</Heading>
      <div className="flex flex-col border-t border-line">
        {letters.map((l) => (
          <details key={l.id} className="group border-b border-line">
            <summary className="press flex min-h-[56px] cursor-pointer list-none items-center gap-3">
              <Icon name="letter" size={20} className="text-gold" />
              <span className="grow text-[15px]">
                <span className="font-semibold">{nameOf(l.author_id)}</span>
                <span className="text-stone"> · {l.opens_on.slice(0, 4)}</span>
              </span>
              <Icon
                name="down"
                size={16}
                className="text-stone transition-transform group-open:rotate-180"
              />
            </summary>
            <div data-selectable className="whitespace-pre-wrap pb-5 text-[16px] leading-[1.7]">
              {l.text}
            </div>
          </details>
        ))}
      </div>
    </section>
  );
}
