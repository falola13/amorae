"use client";

import Link from "next/link";
import { useState } from "react";

import { goalSub, sumOf } from "@/features/together/goals";
import { useGoals } from "@/features/together/hooks";
import { JournalComposer } from "@/features/together/components/journal-composer";
import { MemoryComposer } from "@/features/together/components/memory-composer";

import { Icon, type IconName } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import {
  Alert,
  Bar,
  Button,
  cx,
  EmptyState,
  Initial,
  LinkButton,
  Para,
  Sheet,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { useCouple, useMe } from "@/features/couple/hooks";
import { groupByMonth } from "@/features/timeline/derive";
import { useTimeline } from "@/features/timeline/hooks";
import { dayNum } from "@/lib/dates";
import { routes } from "@/lib/routes";
import type { TimelineFilter, TimelineItem } from "@/lib/api/types";

const FILTERS: { value: TimelineFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "prayer", label: "Prayer" },
  { value: "moments", label: "Moments" },
  { value: "plans", label: "Plans" },
];

const TYPE_ICON: Record<TimelineItem["type"], IconName> = {
  prayer_week: "book",
  prayer_answered: "check",
  memory: "image",
  event: "calendar",
  goal: "target",
  journal: "note",
  appreciation: "heart",
  challenge: "flag",
};

function TimelineRow({
  item,
  photoUrl,
  name,
  last,
}: {
  item: TimelineItem;
  photoUrl?: string | null;
  name?: string;
  last?: boolean;
}) {
  return (
    <Link
      href={item.path}
      className={cx(
        "press flex min-h-[62px] items-center gap-3.5 py-2.5 text-left no-underline",
        !last && "border-b border-line",
      )}
    >
      <Icon name={TYPE_ICON[item.type]} size={22} className="text-stone" />
      <span className="flex min-w-0 grow flex-col gap-px">
        <span className="text-[16px] font-semibold text-ink">{item.title}</span>
        <span className="truncate text-support text-stone">{item.sub}</span>
      </span>
      {item.photo_url ? (
        // eslint-disable-next-line @next/next/no-img-element -- already optimised at the CDN
        <img
          src={item.photo_url}
          alt=""
          className="h-10 w-10 shrink-0 rounded-btn bg-photo object-cover"
        />
      ) : null}
      {item.actor_id ? <Initial letter={name?.[0] ?? "?"} photoUrl={photoUrl} name={name} /> : null}
      <span className="shrink-0 tabular text-support text-stone">{dayNum(item.date)}</span>
    </Link>
  );
}

// Goals still in progress, so what we're working on sits above what we've done.
function WorkingOn() {
  const goals = useGoals();
  const active = (goals.data ?? []).filter((g) => !g.done);
  if (active.length === 0) return null;
  return (
    <section className="mt-4" aria-label="Working on">
      <span className="text-[12px] font-bold uppercase tracking-[0.06em] text-stone">
        Working on
      </span>
      {active.map((g, i) => (
        <Link
          key={g.id}
          href={routes.goal(g.id)}
          className={cx(
            "press flex flex-col gap-2 py-3 text-ink no-underline",
            i < active.length - 1 && "border-b border-line",
          )}
        >
          <span className="text-[16px] font-semibold">{g.title}</span>
          <Bar pct={(sumOf(g) / g.target) * 100} label={goalSub(g)} />
          <span className="text-support text-stone">{goalSub(g)}</span>
        </Link>
      ))}
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
          <Title>Our story</Title>
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
        <Para className="mt-1.5">Everything you’ve done together, newest first.</Para>
        <div role="tablist" aria-label="Filter" className="-mx-1 mt-3 flex flex-wrap gap-1">
          {FILTERS.map((f) => (
            <button
              key={f.value}
              type="button"
              role="tab"
              aria-selected={filter === f.value}
              onClick={() => setFilter(f.value)}
              className={cx(
                "press h-9 rounded-full px-3.5 text-[14px] font-semibold",
                filter === f.value ? "bg-plum-tint text-plum" : "text-stone",
              )}
            >
              {f.label}
            </button>
          ))}
        </div>
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
              <div className="mt-2 flex flex-col pb-4">
                {groups.map((g, gi) => (
                  <div key={`${g.month}-${gi}`}>
                    <div className="pb-1.5 pt-[18px]">
                      <span className="text-[12px] font-bold uppercase tracking-[0.06em] text-stone">
                        {g.month}
                      </span>
                    </div>
                    {g.items.map((item, i) => {
                      const who = actor(item.actor_id);
                      return (
                        <TimelineRow
                          key={item.id}
                          item={item}
                          name={who.name}
                          photoUrl={who.photoUrl}
                          last={gi === groups.length - 1 && i === g.items.length - 1}
                        />
                      );
                    })}
                  </div>
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
