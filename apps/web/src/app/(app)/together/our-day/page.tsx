"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import {
  Button,
  Initial,
  LinkButton,
  LoadProblem,
  Micro,
  Ornament,
  Para,
  Section,
  Skeleton,
  TopBar,
  cx,
} from "@/components/ui/kit";
import { Scripture } from "@/components/ui/scripture";
import {
  count,
  daysTogether,
  notesOfTheYear,
  photosOfTheYear,
  whenLabel,
  yearTiles,
  yearsWord,
} from "@/features/anniversary/derive";
import { useAnniversary } from "@/features/anniversary/hooks";
import { Opening, useOccasionTheme } from "@/features/anniversary/occasion";
import { useCouple } from "@/features/couple/hooks";
import { useAppreciations, useMemories } from "@/features/together/hooks";
import { isAbsence } from "@/lib/api/envelope";
import type { Anniversary, AnniversaryLetter, Couple } from "@/lib/api/types";
import { iso, longDate, longDateYear, shortMonth, weekdayDate } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";

export default function OurDayPage() {
  return (
    <Suspense fallback={null}>
      <OurDay />
    </Suspense>
  );
}

/**
 * The anniversary's own page. In the days before, it is where the letters
 * are written and the day is looked forward to; on the day, it is the day:
 * the years, the letters opened, the year they had, and a blessing. Anyone
 * can see the day ahead of time with ?preview=1 — everything but the other
 * one's letter, which the server keeps sealed regardless.
 */
function OurDay() {
  const preview = useSearchParams().get("preview") === "1";
  const anniversary = useAnniversary();
  const couple = useCouple();
  const a = anniversary.data;
  const theDay = a?.days_until === 0;
  const showDay = theDay || preview;
  useOccasionTheme(showDay && !!a);

  return (
    <>
      <TopBar backHref={routes.home} back="Home" />
      <Main className="pb-10">
        {anniversary.isPending || !couple.data ? (
          <div className="pt-6">
            <Skeleton lines={4} />
          </div>
        ) : anniversary.isError && isAbsence(anniversary.error) ? (
          <NoAnniversary />
        ) : anniversary.isError || !a ? (
          <LoadProblem
            icon="alert"
            tone="error"
            title="Your day didn’t load"
            text="Nothing is lost."
            onRetry={() => void anniversary.refetch()}
          />
        ) : (
          <Day a={a} couple={couple.data} showDay={showDay} preview={preview && !theDay} />
        )}
      </Main>
    </>
  );
}

function NoAnniversary() {
  return (
    <div className="flex grow flex-col justify-center gap-3.5 pb-12">
      <Icon name="heart" size={28} strokeWidth={1.4} className="text-plum" />
      <h1 className="m-0 text-[26px] font-semibold leading-tight tracking-[-0.02em]">
        Your anniversary will live here
      </h1>
      <Para size="lg">
        Once you’ve been together a year, this page becomes your day: letters to each other, and the
        year you had. Add the day you started if it isn’t set.
      </Para>
      <div className="mt-3">
        <LinkButton href={routes.settingsCouple} variant="secondary">
          Set the day you started
        </LinkButton>
      </div>
    </div>
  );
}

function Day({
  a,
  couple,
  showDay,
  preview,
}: {
  a: Anniversary;
  couple: Couple;
  showDay: boolean;
  preview: boolean;
}) {
  const [replay, setReplay] = useState(false);
  const me = couple.me;
  const partner = couple.partner?.display_name ?? "your partner";
  const first = partner.split(" ")[0];
  const nameOf = (id: string) => (id === me.id ? me.display_name : partner);

  return (
    <>
      {preview ? (
        <div className="mt-2 flex items-start gap-3 rounded-card border border-dashed border-edge px-4 py-3">
          <Icon name="moon" size={20} className="mt-0.5 shrink-0 text-stone" />
          <div className="flex grow flex-col">
            <span className="text-[15px] font-semibold">A preview of {weekdayDate(a.on)}</span>
            <span className="text-support text-stone">
              Only you are seeing this. {first}&rsquo;s letter stays sealed until the day.
            </span>
          </div>
        </div>
      ) : null}

      <Hero a={a} couple={couple} showDay={showDay} />

      <Letters a={a} me={me.display_name} partner={partner} showDay={showDay} />

      {showDay ? <YearInNumbers a={a} /> : null}
      {showDay ? <Photos a={a} /> : null}
      {showDay ? <Notes a={a} nameOf={nameOf} /> : null}

      {a.past.length > 0 ? <PastLetters letters={a.past} nameOf={nameOf} /> : null}

      {showDay ? (
        <>
          <div className="mt-10">
            <Scripture
              reference="Ecclesiastes 4:9, 12"
              verse="Two are better than one; because they have a good reward for their labour… and a threefold cord is not quickly broken."
            />
          </div>
          <div className="mt-8 flex flex-col gap-1">
            <LinkButton href={routes.memories} icon="image">
              Keep a memory of today
            </LinkButton>
            <Button variant="text" onClick={() => setReplay(true)}>
              {preview ? "See the opening" : "Play the opening again"}
            </Button>
          </div>
          <Ornament caption={`${yearsWord(a.years)}, and counting.`} />
        </>
      ) : null}

      {replay ? <Opening on={a.on} years={a.years} onClose={() => setReplay(false)} /> : null}
    </>
  );
}

function Hero({ a, couple, showDay }: { a: Anniversary; couple: Couple; showDay: boolean }) {
  const partner = couple.partner;
  const started = couple.started_on;
  // On a preview the day is still ahead, so count to it rather than to today.
  const days = started ? daysTogether(started, showDay ? a.on : iso(today())) : null;
  return (
    <section className="flex animate-rise flex-col items-center pb-2 pt-8 text-center">
      <Micro tone="plum" className="tracking-[0.16em]">
        {showDay ? longDateYear(a.on) : weekdayDate(a.on)}
      </Micro>
      <h1 className="m-0 mt-3 text-[54px] font-semibold leading-[1.02] tracking-[-0.035em] text-plum-dark">
        {yearsWord(a.years)}
      </h1>
      {showDay ? (
        <Para size="lg" className="mt-3">
          Happy anniversary.
          {started ? <> Together since {longDateYear(started)}.</> : null}
        </Para>
      ) : (
        <Para size="lg" className="mt-3">
          {whenLabel(a.days_until) === "tomorrow" ? "Tomorrow" : `In ${a.days_until} days`}.{" "}
          {a.mine
            ? "Your letter is sealed until then."
            : a.writable
              ? `Write ${partner?.display_name.split(" ")[0] ?? "them"} something to open on the day.`
              : "Something to look forward to."}
        </Para>
      )}
      <div className="mt-6 flex items-center gap-3">
        <Initial
          letter={couple.me.display_name[0]}
          photoUrl={couple.me.photo_url}
          name={couple.me.display_name}
          size={44}
        />
        <Icon name="heart" size={20} className="text-plum" />
        {partner ? (
          <Initial
            letter={partner.display_name[0]}
            photoUrl={partner.photo_url}
            name={partner.display_name}
            size={44}
          />
        ) : null}
      </div>
      {days ? (
        <p className="m-0 mt-4 text-support text-stone">
          {count(days)} days {showDay ? "of choosing each other" : "together so far"}
        </p>
      ) : null}
    </section>
  );
}

function Letters({
  a,
  me,
  partner,
  showDay,
}: {
  a: Anniversary;
  me: string;
  partner: string;
  showDay: boolean;
}) {
  const first = partner.split(" ")[0];
  return (
    <Section label="Letters" className="mt-8">
      <div className="flex flex-col gap-3 pt-2">
        {a.theirs ? (
          <OpenLetter letter={a.theirs} from={partner} to={me} />
        ) : a.theirs_sealed ? (
          <div className="flex items-center gap-4 rounded-card bg-occasion px-5 py-5">
            <Envelope sealed />
            <div className="flex flex-col">
              <span className="text-[17px] font-semibold">{first} has written you something</span>
              <span className="text-support text-stone">
                Sealed until {a.days_until === 1 ? "tomorrow" : weekdayDate(a.on)}.
              </span>
            </div>
          </div>
        ) : showDay ? (
          <p className="m-0 text-support text-stone">
            If {first} writes you one today, it will be here.
          </p>
        ) : null}

        {a.mine ? (
          <Link
            href={routes.ourDayLetter}
            className="press flex items-center gap-4 rounded-card border border-line bg-surface px-5 py-4 text-ink no-underline"
          >
            <Icon name="letter" size={22} className="shrink-0 text-plum" />
            <span className="flex grow flex-col">
              <span className="text-[16px] font-semibold">Your letter to {first}</span>
              <span className="text-support text-stone">
                {a.days_until === 0
                  ? `${first} can read it now`
                  : `Sealed until ${a.days_until === 1 ? "tomorrow" : longDate(a.on)} · tap to change it`}
              </span>
            </span>
            <Icon name="right" size={18} className="text-stone" />
          </Link>
        ) : a.writable ? (
          <Link
            href={routes.ourDayLetter}
            className="press flex items-center gap-4 rounded-card border border-dashed border-edge px-5 py-4 text-ink no-underline"
          >
            <Icon name="pencil" size={22} className="shrink-0 text-plum" />
            <span className="flex grow flex-col">
              <span className="text-[16px] font-semibold">Write {first} a letter</span>
              <span className="text-support text-stone">
                {a.days_until === 0
                  ? "They can read it as soon as you keep it."
                  : `Kept sealed until ${a.days_until === 1 ? "tomorrow" : weekdayDate(a.on)}.`}
              </span>
            </span>
            <Icon name="right" size={18} className="text-stone" />
          </Link>
        ) : (
          <p className="m-0 text-support text-stone">
            Letters for this one open for writing on {weekdayDate(a.writable_from)}.
          </p>
        )}
      </div>
    </Section>
  );
}

/** A small drawn envelope; the seal is a heart. */
function Envelope({ sealed }: { sealed?: boolean }) {
  return (
    <svg width="48" height="36" viewBox="0 0 48 36" aria-hidden="true" className="shrink-0">
      <rect
        x="1"
        y="1"
        width="46"
        height="34"
        rx="4"
        fill="var(--color-surface)"
        stroke="var(--color-edge)"
        strokeWidth="1.2"
      />
      <path d="M2 3l22 16L46 3" fill="none" stroke="var(--color-edge)" strokeWidth="1.2" />
      {sealed ? (
        <path
          d="M24 25.5c-3.4-2.3-5-4.4-5-6.3 0-1.6 1.3-2.9 2.8-2.9.9 0 1.7.5 2.2 1.2.5-.7 1.3-1.2 2.2-1.2 1.5 0 2.8 1.3 2.8 2.9 0 1.9-1.6 4-5 6.3z"
          fill="var(--color-plum)"
        />
      ) : null}
    </svg>
  );
}

/** Their letter, opened by a tap — the one moment on the page worth a gesture. */
function OpenLetter({ letter, from, to }: { letter: AnniversaryLetter; from: string; to: string }) {
  const [open, setOpen] = useState(false);
  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="press flex w-full items-center gap-4 rounded-card bg-occasion px-5 py-5 text-left"
      >
        <Envelope sealed />
        <span className="flex grow flex-col">
          <span className="text-[17px] font-semibold">A letter from {from.split(" ")[0]}</span>
          <span className="text-support text-plum">Tap to open it</span>
        </span>
      </button>
    );
  }
  return (
    <article
      data-selectable
      className="animate-unfold rounded-card border border-line bg-letter bg-position-[0_24px] px-6 pb-7 pt-6 shadow-[0_18px_40px_-28px_rgba(60,20,40,0.45)]"
    >
      <p className="m-0 text-[19px] font-semibold leading-[32px]">Dear {to.split(" ")[0]},</p>
      <div className="mt-2 whitespace-pre-wrap text-[17px] leading-[32px] text-ink">
        {letter.text}
      </div>
      <p className="m-0 mt-4 text-right text-[17px] font-semibold leading-[32px] text-plum">
        — {from.split(" ")[0]}
      </p>
    </article>
  );
}

function YearInNumbers({ a }: { a: Anniversary }) {
  const tiles = yearTiles(a.year);
  if (tiles.length === 0) return null;
  return (
    <Section
      label="Your year"
      className="mt-10"
      trailing={
        <span className="text-[13px] text-stone">
          {shortMonth(a.year.from)} {a.year.from.slice(0, 4)} – {shortMonth(a.year.to)}{" "}
          {a.year.to.slice(0, 4)}
        </span>
      }
    >
      <div className="grid grid-cols-2 gap-2.5 pt-2">
        {tiles.map((t, i) => (
          <div
            key={t.label}
            className={cx(
              "flex flex-col rounded-card px-4 py-4",
              i === 0 ? "col-span-2 bg-occasion" : "border border-line bg-surface",
            )}
          >
            <span
              className={cx(
                "tabular font-semibold leading-none tracking-[-0.03em] text-plum-dark",
                i === 0 ? "text-[44px]" : "text-[30px]",
              )}
            >
              {count(t.n)}
            </span>
            <span className="mt-1.5 text-support text-stone">{t.label}</span>
          </div>
        ))}
      </div>
    </Section>
  );
}

function Photos({ a }: { a: Anniversary }) {
  const memories = useMemories();
  const photos = photosOfTheYear(memories.data ?? [], a.year);
  if (photos.length === 0) return null;
  return (
    <Section label="Moments you kept" className="mt-10">
      <div className="grid grid-cols-3 gap-1.5 pt-2">
        {photos.map((m, i) => (
          <Link
            key={m.id}
            href={routes.memories}
            className={cx(
              "press relative block overflow-hidden rounded-input bg-photo",
              // One large only where it leaves no hole: three or six fill the grid exactly.
              i === 0 && photos.length % 3 === 0 ? "col-span-2 row-span-2" : "aspect-square",
            )}
          >
            {/* eslint-disable-next-line @next/next/no-img-element -- already optimised at the CDN */}
            <img src={m.photo_url} alt={m.title} className="h-full w-full object-cover" />
            <span className="absolute inset-x-0 bottom-0 bg-linear-to-t from-black/55 to-transparent px-2 pb-1.5 pt-6 text-[12px] font-semibold text-white">
              {m.title}
            </span>
          </Link>
        ))}
      </div>
    </Section>
  );
}

function Notes({ a, nameOf }: { a: Anniversary; nameOf: (id: string) => string }) {
  const notes = useAppreciations();
  const picked = notesOfTheYear(notes.data ?? [], a.year);
  if (picked.length === 0) return null;
  return (
    <Section label="Things you said" className="mt-10">
      <div className="flex flex-col gap-5 pt-3">
        {picked.map((n) => (
          <figure key={n.id} className="m-0 border-l-2 border-plum pl-4">
            <blockquote data-selectable className="m-0 text-[18px] leading-[1.5] text-ink">
              &ldquo;{n.text}&rdquo;
            </blockquote>
            <figcaption className="mt-1 text-support text-stone">
              {nameOf(n.from_id).split(" ")[0]} · {longDateYear(n.date)}
            </figcaption>
          </figure>
        ))}
      </div>
    </Section>
  );
}

function PastLetters({
  letters,
  nameOf,
}: {
  letters: AnniversaryLetter[];
  nameOf: (id: string) => string;
}) {
  return (
    <Section label="Letters from other years" className="mt-10">
      <div className="flex flex-col border-t border-line">
        {letters.map((l) => (
          <details key={l.id} className="group border-b border-line">
            <summary className="press flex min-h-[56px] cursor-pointer list-none items-center gap-3">
              <Icon name="letter" size={20} className="text-stone" />
              <span className="grow text-[15px]">
                <span className="font-semibold">{nameOf(l.author_id).split(" ")[0]}</span>
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
    </Section>
  );
}
