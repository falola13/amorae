"use client";

import Link from "next/link";
import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";

import { goalSub, sumOf } from "@/features/together/goals";
import { useGoals, useJournal } from "@/features/together/hooks";
import { JournalComposer } from "@/features/together/components/journal-composer";
import { MemoryComposer } from "@/features/together/components/memory-composer";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import {
  Alert,
  Bar,
  Button,
  cx,
  EmptyState,
  LinkButton,
  Para,
  Sheet,
  Skeleton,
  Tabs,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { useCouple, useMe } from "@/features/couple/hooks";
import { groupByMonth } from "@/features/timeline/derive";
import { useTimeline, useTimelineMonths } from "@/features/timeline/hooks";
import { dayName, dayNum, iso, longDate, monthLabel, monthName } from "@/lib/dates";
import { today } from "@/lib/today";
import { ordinal, yearsWord } from "@/features/anniversary/derive";
import { routes } from "@/lib/routes";
import type { TimelineFilter, TimelineItem } from "@/lib/api/types";

const FILTERS: { value: TimelineFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "prayer", label: "Prayer" },
  { value: "memories", label: "Memories" },
  { value: "journal", label: "Journal" },
  { value: "notes", label: "Notes" },
  { value: "questions", label: "Questions" },
  { value: "plans", label: "Plans" },
];

// Each entry is named by a small word rather than an icon: the words do the work.
const KIND: Record<TimelineItem["type"], string> = {
  prayer_week: "Prayer",
  prayer_answered: "Prayer answered",
  memory: "Memory",
  event: "Plan",
  goal: "Goal",
  journal: "Journal",
  appreciation: "Appreciation",
  challenge: "Challenge",
  anniversary: "Anniversary",
  question: "Question",
};

// The feed sends the first 80 characters of longer text; say so rather than
// stopping mid-word.
const EXCERPT = 80;
function excerpt(s: string): string {
  if (s.length < EXCERPT) return s;
  const cut = s.slice(0, s.lastIndexOf(" ") > 40 ? s.lastIndexOf(" ") : s.length);
  return `${cut.replace(/[\s,.;:—-]+$/, "")}…`;
}

/** Our story's own marker for each anniversary that has come round, placed
 *  on its day among what you did, so the day is easy to find again. Only
 *  within what's been loaded, unless the whole story has been. */
function withAnniversaries(
  items: TimelineItem[],
  startedOn: string | undefined,
  complete: boolean,
  until?: string,
): TimelineItem[] {
  if (!startedOn || items.length === 0) return items;
  const [y, m, d] = startedOn.split("-").map(Number);
  const todayIso = iso(today());
  const oldest = items[items.length - 1].date;
  const marks: TimelineItem[] = [];
  for (let year = y + 1; ; year++) {
    const day = `${year}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
    if (day > todayIso) break;
    // Jumped to a month: nothing newer than its last day belongs above it.
    if (until && day > until) break;
    if (!complete && day < oldest) continue;
    const n = year - y;
    marks.push({
      id: `anniversary-${year}`,
      type: "anniversary",
      at: `${day}T23:59:59Z`,
      date: day,
      title: `Our ${ordinal(n)} anniversary`,
      sub: `${yearsWord(n)} together · your letters`,
      path: routes.ourDay(),
    });
  }
  if (marks.length === 0) return items;
  // Stable sort: the day's own entries keep their order, the marker heads them.
  return [...items, ...marks].sort((a, b) => {
    if (a.date !== b.date) return a.date < b.date ? 1 : -1;
    if (a.type === "anniversary") return -1;
    return b.type === "anniversary" ? 1 : 0;
  });
}

/** A journal entry read in full, since the feed carries only its opening. */
function JournalEntrySheet({
  item,
  name,
  onClose,
}: {
  item: TimelineItem;
  name?: string;
  onClose: () => void;
}) {
  const journal = useJournal();
  const entry = journal.data?.find((j) => j.id === item.id);
  const date = entry?.date ?? item.date;
  return (
    <Sheet open onClose={onClose} title="From the journal" labelledBy={`journal-${item.id}-h`}>
      <div className="flex items-baseline justify-between gap-3 text-support text-stone">
        <span>
          {name ? <span className="font-semibold text-ink">{name}</span> : null}
          {name ? " · " : ""}
          {longDate(date)} {date.slice(0, 4)}
        </span>
        <span className="text-micro uppercase">{entry?.tag ?? item.title}</span>
      </div>
      {journal.isPending ? (
        <Skeleton lines={4} />
      ) : (
        <p className="m-0 whitespace-pre-wrap font-text text-bodylg leading-[1.6]" data-selectable>
          {entry?.text ?? item.sub}
        </p>
      )}
      <LinkButton href={routes.journal} variant="secondary">
        Open the journal
      </LinkButton>
    </Sheet>
  );
}

/** An entry hung off its date in the left margin. Rules start at the text
 *  column, not the screen edge, and a photograph takes the whole column. */
function TimelineRow({
  item,
  name,
  first,
}: {
  item: TimelineItem;
  name?: string;
  first?: boolean;
}) {
  // Journal and appreciation are the couple's own words, so they're set as
  // words in the serif; their "title" is only the tag or the kind again.
  const words = item.type === "journal" || item.type === "appreciation";
  const [reading, setReading] = useState(false);
  const rowClass = "press grid grid-cols-[44px_1fr] gap-x-1 py-5 text-left text-ink no-underline";
  const body = (
    <>
      <span className="lining pt-px font-text text-[22px] leading-none">
        {String(dayNum(item.date)).padStart(2, "0")}
        <span className="mt-1 block font-sans text-micro uppercase text-stone">
          {dayName(item.date).slice(0, 3)}
        </span>
      </span>
      <span className="relative flex min-w-0 flex-col">
        {first ? null : (
          <span aria-hidden="true" className="absolute inset-x-0 -top-5 border-t border-line" />
        )}
        {item.photo_url ? (
          // eslint-disable-next-line @next/next/no-img-element -- already optimised at the CDN
          <img
            src={item.photo_url}
            alt=""
            loading="lazy"
            decoding="async"
            className="mb-3 aspect-[4/3] w-full rounded-xs bg-photo object-cover"
          />
        ) : null}
        <span
          className={cx(
            "text-support font-medium",
            item.type === "prayer_answered" ? "text-green" : "text-stone",
          )}
        >
          {KIND[item.type]}
          {name ? ` · ${name}` : ""}
        </span>
        {words ? null : (
          <span className="mt-0.5 font-text text-[19px] leading-[1.3]">{item.title}</span>
        )}
        {item.sub ? (
          words ? (
            <span
              className={cx(
                "mt-1 font-text text-bodylg leading-[1.45]",
                item.type === "appreciation" && "italic",
              )}
            >
              {item.type === "appreciation" ? `“${excerpt(item.sub)}”` : excerpt(item.sub)}
            </span>
          ) : (
            <span className="text-support text-stone">{item.sub}</span>
          )
        ) : null}
      </span>
    </>
  );
  // A journal entry opens in place; everything else goes where it belongs.
  if (item.type === "journal") {
    return (
      <>
        <button type="button" onClick={() => setReading(true)} className={cx(rowClass, "w-full")}>
          {body}
        </button>
        {reading ? (
          <JournalEntrySheet item={item} name={name} onClose={() => setReading(false)} />
        ) : null}
      </>
    );
  }
  return (
    <Link href={item.path} className={rowClass}>
      {body}
    </Link>
  );
}

// Goals still in progress, so what we're working on sits above what we've done.
function WorkingOn() {
  const goals = useGoals();
  const active = (goals.data ?? []).filter((g) => !g.done);
  if (active.length === 0) return null;
  // Ruled newspaper columns, not tiles: a hairline between goals, and the
  // strip runs off the edge because it scrolls.
  return (
    <section className="mt-6" aria-label="Working on">
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="m-0 font-text text-section italic">Working on</h2>
      </div>
      <div className="-mr-6 mt-3 flex overflow-x-auto md:-mr-8">
        {active.map((g, i) => (
          <Link
            key={g.id}
            href={routes.goal(g.id)}
            className={cx(
              "press flex w-[150px] shrink-0 flex-col py-1 pr-4 text-ink no-underline",
              i > 0 && "border-l border-line pl-4",
            )}
          >
            <span className="mb-3 font-text text-bodylg leading-[1.25]">{g.title}</span>
            <span>
              <Bar pct={(sumOf(g) / g.target) * 100} label={goalSub(g)} />
            </span>
            <span className="lining mt-2 text-support text-stone">{goalSub(g)}</span>
          </Link>
        ))}
      </div>
    </section>
  );
}

/** Every month that has something in it for this filter, by year. */
function JumpSheet({
  open,
  filter,
  current,
  onClose,
  onPick,
}: {
  open: boolean;
  filter: TimelineFilter;
  current?: string;
  onClose: () => void;
  onPick: (month: string) => void;
}) {
  const months = useTimelineMonths(filter, open);
  const years: { year: string; months: { month: string; count: number }[] }[] = [];
  for (const m of months.data ?? []) {
    const year = m.month.slice(0, 4);
    const last = years[years.length - 1];
    if (last && last.year === year) last.months.push(m);
    else years.push({ year, months: [m] });
  }
  return (
    <Sheet open={open} onClose={onClose} title="Jump to" labelledBy="story-jump-h">
      {months.isPending ? (
        <Skeleton lines={4} />
      ) : years.length === 0 ? (
        <Para size="support">Nothing here yet.</Para>
      ) : (
        <div className="flex flex-col">
          {years.map((y) => (
            <section key={y.year} aria-label={y.year}>
              <h3 className="m-0 mt-2 font-text text-title">{y.year}</h3>
              {y.months.map((m) => (
                <button
                  key={m.month}
                  type="button"
                  onClick={() => onPick(m.month)}
                  aria-current={m.month === current ? "true" : undefined}
                  className={cx(
                    "press flex h-11 w-full items-center justify-between border-b border-line text-left text-body",
                    m.month === current ? "font-semibold text-plum" : "text-ink",
                  )}
                >
                  <span>{monthName(`${m.month}-01`)}</span>
                  <span className="lining text-support text-stone">{m.count}</span>
                </button>
              ))}
            </section>
          ))}
        </div>
      )}
    </Sheet>
  );
}

export default function HistoryPage() {
  return (
    <Suspense fallback={null}>
      <History />
    </Suspense>
  );
}

function History() {
  // ?filter=prayer opens straight onto past prayer weeks (the Prayers page links here).
  const asked = useSearchParams().get("filter");
  const [filter, setFilter] = useState<TimelineFilter>(
    FILTERS.some((f) => f.value === asked) ? (asked as TimelineFilter) : "all",
  );
  // Set by "Jump to": starts the feed at the end of that month (YYYY-MM).
  const [month, setMonth] = useState<string | undefined>();
  const [jumping, setJumping] = useState(false);
  const timeline = useTimeline(filter, month);
  const couple = useCouple();
  const me = useMe();
  const [adding, setAdding] = useState(false);
  const [memoryOpen, setMemoryOpen] = useState(false);
  const [journalOpen, setJournalOpen] = useState(false);
  // Persists (no auto-dismiss) since it flags a photo that still needs re-uploading.
  const [photoFailed, setPhotoFailed] = useState(false);

  const meId = me.data?.id;
  const partnerId = couple.data?.partner?.id;
  const actor = (id?: string) => {
    if (id === meId) return { name: me.data?.display_name, photoUrl: me.data?.photo_url };
    if (id === partnerId)
      return {
        name: couple.data?.partner?.display_name,
        photoUrl: couple.data?.partner?.photo_url,
      };
    return { name: undefined, photoUrl: undefined };
  };

  return (
    <>
      {/* A tab of its own, so nothing to go back to; the bar only carries the add button. */}
      <TopBar
        right={
          <button
            type="button"
            aria-label="Add to our story"
            onClick={() => setAdding(true)}
            className="press -mr-1 flex h-11 w-11 items-center justify-center text-ink"
          >
            <Icon name="plus" size={24} />
          </button>
        }
      />
      <Main>
        <div className="pt-1.5">
          <Title size="display">Our story</Title>
        </div>
        {photoFailed ? (
          <div className="mt-3 flex flex-col gap-1.5">
            <Alert message="The moment was saved, but the photo didn’t upload." />
            <button
              type="button"
              onClick={() => setPhotoFailed(false)}
              className="press self-start px-1 text-[15px] font-semibold text-plum"
            >
              Got it
            </button>
          </div>
        ) : null}
        <Para size="support" className="mt-2">
          Everything you’ve done together, newest first.
        </Para>
        <Tabs
          label="Filter"
          options={FILTERS}
          value={filter}
          onChange={(f) => {
            setFilter(f);
            setMonth(undefined);
          }}
          className="-mx-6 mt-4 px-3.5 md:-mx-8 md:px-5.5"
        />
        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => setJumping(true)}
            className="press -mr-2 flex h-11 items-center px-2 text-support font-semibold text-plum"
          >
            Jump to
          </button>
        </div>
        {month ? (
          <div className="flex items-center justify-between gap-3 border-t border-line pt-1 text-support text-stone">
            <span>From {monthLabel(`${month}-01`)}</span>
            <button
              type="button"
              onClick={() => setMonth(undefined)}
              className="press -mr-2 flex h-11 items-center px-2 font-semibold text-plum"
            >
              Back to newest
            </button>
          </div>
        ) : null}
        {/* Only where goals belong: under "Prayer" it would be out of place. */}
        {filter === "all" || filter === "plans" ? <WorkingOn /> : null}
        <QueryState
          queries={[timeline]}
          frame={inPage}
          loading={
            <div className="pt-4">
              <Skeleton lines={4} />
            </div>
          }
        >
          {(data) => {
            const fetched = data.pages.flatMap((p) => p.items);
            const items =
              filter === "all" || filter === "memories"
                ? withAnniversaries(
                    fetched,
                    couple.data?.started_on,
                    !timeline.hasNextPage,
                    month ? `${month}-31` : undefined,
                  )
                : fetched;
            if (items.length === 0) {
              return (
                <div className="pt-4">
                  <EmptyState
                    ghost="dots"
                    title="Nothing here yet"
                    text="Your story fills in as you go."
                    cta={
                      <LinkButton href={routes.together} variant="secondary">
                        Back to Together
                      </LinkButton>
                    }
                  />
                </div>
              );
            }
            const groups = groupByMonth(items);
            return (
              <div className="flex flex-col pb-4">
                {groups.map((g, gi) => (
                  // A chapter per month: a heavy ink rule and the month in the serif.
                  <section key={`${g.month}-${gi}`} className="mt-10">
                    <div className="flex items-baseline justify-between border-t-[1.5px] border-ink pt-3">
                      <h2 className="m-0 font-text text-title">{monthName(g.items[0].date)}</h2>
                      <span className="lining text-support text-stone">
                        {g.items[0].date.slice(0, 4)}
                      </span>
                    </div>
                    {g.items.map((item, i) => (
                      <TimelineRow
                        key={item.id}
                        item={item}
                        name={actor(item.actor_id).name}
                        first={i === 0}
                      />
                    ))}
                  </section>
                ))}
                {timeline.hasNextPage ? (
                  <div className="pt-4">
                    <Button
                      variant="secondary"
                      loading={timeline.isFetchingNextPage}
                      onClick={() => timeline.fetchNextPage()}
                    >
                      Show earlier
                    </Button>
                  </div>
                ) : null}
              </div>
            );
          }}
        </QueryState>
      </Main>
      <JumpSheet
        open={jumping}
        filter={filter}
        current={month}
        onClose={() => setJumping(false)}
        onPick={(m) => {
          setMonth(m);
          setJumping(false);
        }}
      />
      <Sheet
        open={adding}
        onClose={() => setAdding(false)}
        title="Add to our story"
        labelledBy="story-add-h"
      >
        <div className="flex flex-col gap-1">
          <Button
            variant="secondary"
            icon="image"
            onClick={() => {
              setAdding(false);
              setMemoryOpen(true);
            }}
          >
            Add a memory
          </Button>
          <Button
            variant="secondary"
            icon="pencil"
            onClick={() => {
              setAdding(false);
              setJournalOpen(true);
            }}
          >
            Write in the journal
          </Button>
          <LinkButton href={routes.goalNew} variant="secondary" icon="target">
            Start a goal
          </LinkButton>
        </div>
      </Sheet>
      <MemoryComposer
        open={memoryOpen}
        onClose={() => setMemoryOpen(false)}
        onPhotoFailed={() => setPhotoFailed(true)}
      />
      <JournalComposer open={journalOpen} entry={null} onClose={() => setJournalOpen(false)} />
    </>
  );
}
