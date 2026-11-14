"use client";

import Link from "next/link";
import { useState } from "react";

import { goalSub, sumOf } from "@/features/together/goals";
import { useGoals } from "@/features/together/hooks";
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
import { useTimeline } from "@/features/timeline/hooks";
import { dayName, dayNum, monthName } from "@/lib/dates";
import { routes } from "@/lib/routes";
import type { TimelineFilter, TimelineItem } from "@/lib/api/types";

const FILTERS: { value: TimelineFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "prayer", label: "Prayer" },
  { value: "moments", label: "Moments" },
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
};

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
  return (
    <Link
      href={item.path}
      className="press grid grid-cols-[44px_1fr] gap-x-1 py-5 text-left text-ink no-underline"
    >
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
        <span className="mt-0.5 font-text text-[19px] leading-[1.3]">{item.title}</span>
        {item.sub ? <span className="text-support text-stone">{item.sub}</span> : null}
      </span>
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
              "press flex min-h-[124px] w-[150px] shrink-0 flex-col py-1 pr-4 text-ink no-underline",
              i > 0 && "border-l border-line pl-4",
            )}
          >
            <span className="mb-4 font-text text-bodylg leading-[1.25]">{g.title}</span>
            <span className="mt-auto">
              <Bar pct={(sumOf(g) / g.target) * 100} label={goalSub(g)} />
            </span>
            <span className="lining mt-2 text-support text-stone">{goalSub(g)}</span>
          </Link>
        ))}
      </div>
    </section>
  );
}

export default function History() {
  const [filter, setFilter] = useState<TimelineFilter>("all");
  const timeline = useTimeline(filter);
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
          onChange={setFilter}
          className="-mx-6 mt-4 px-3.5 md:-mx-8 md:px-5.5"
        />
        {/* Only on the whole story: under "Prayer" it would be out of place. */}
        {filter === "all" ? <WorkingOn /> : null}
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
            const items = data.pages.flatMap((p) => p.items);
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
