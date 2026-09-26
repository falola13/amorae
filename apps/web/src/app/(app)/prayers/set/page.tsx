"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useCouple } from "@/features/couple/hooks";
import { daysLabel } from "@/features/prayers/derive";
import { useHistory, usePublish, useSavePoints, useWeek } from "@/features/prayers/hooks";
import { useDragReorder } from "@/features/prayers/use-drag-reorder";
import { range } from "@/lib/dates";
import { routes } from "@/lib/routes";
import type { PrayerPoint } from "@/lib/api/types";
import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { Button, cx, LinkButton, Micro, Para, Sheet, Skeleton } from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";

const MAX = 10;

/** Setter's list: reorder by drag handle (pointer) or the up/down keys on a focused handle. */
export default function SetPrayers() {
  const week = useWeek();
  const couple = useCouple();
  const save = useSavePoints();
  const publish = usePublish();
  const router = useRouter();
  const [confirm, setConfirm] = useState(false);
  const [saved, setSaved] = useState(true);
  // Offers the most recent non-empty week to copy from (Q-26).
  const history = useHistory();
  const previous = history.data?.find((h) => h.points.length > 0);
  const partner = couple.data?.partner?.display_name ?? "your partner";
  const commit = (next: PrayerPoint[]) => {
    setSaved(false);
    save.mutate(next, { onSuccess: () => setSaved(true) });
  };
  // Ids are dropped so the server creates new rows instead of reusing last week's.
  const startFromLast = () => {
    if (!previous) return;
    commit(previous.points.map((p) => ({ ...p, id: "" })));
  };
  const drag = useDragReorder(week.data?.points ?? null, commit);

  return (
    <QueryState
      queries={[week]}
      frame={inPage}
      loading={
        <Main>
          <div className="pt-4">
            <Skeleton />
          </div>
        </Main>
      }
    >
      {(w) => {
        // Either of you can edit or publish now — turns just say who started the week.
        const me = couple.data?.me.id;
        const mine = !me || w.setter_id === me;
        const items = drag.items ?? w.points;
        const shared = w.status === "published";
        const doPublish = () =>
          publish.mutate(undefined, {
            onSuccess: () => router.replace(routes.home),
            onQueued: () => router.replace(routes.home),
          });
        return (
          <>
            <div className="flex h-11 shrink-0 items-center justify-between px-3">
              <Link
                href={routes.home}
                aria-label="Close"
                className="press flex h-11 w-11 items-center justify-center text-ink"
              >
                <Icon name="x" size={22} />
              </Link>
              <div role="status" className="flex items-center gap-1.5 pr-3 text-[13px] text-stone">
                {saved ? (
                  <>
                    <Icon name="check" size={16} strokeWidth={2} className="text-green" />
                    {shared ? "Saved" : "Draft saved"}
                  </>
                ) : (
                  "Saving…"
                )}
              </div>
            </div>
            <Main>
              <div className="pt-3">
                <Micro>{range(w.week_start, w.week_end)}</Micro>
              </div>
              <h1 className="m-0 mt-2 text-[30px] font-semibold leading-[1.18] tracking-[-0.022em]">
                {mine ? "It’s your week." : "This week’s prayers"}
              </h1>
              <Para className="mt-1.5">
                {!mine
                  ? `It’s ${partner}’s turn this week — you can still write it.`
                  : shared
                    ? `${partner} can see these. Add another whenever you think of one, and change any they haven’t prayed yet.`
                    : `Write what’s on your heart. One is enough, and you can have up to ${MAX}.`}
              </Para>
              {items.length === 0 && previous ? (
                <button
                  type="button"
                  onClick={startFromLast}
                  disabled={save.isPending}
                  className="press mt-4 flex w-full items-center gap-3 rounded-input border border-edge px-4 py-3 text-left"
                >
                  <Icon name="copy" size={20} className="shrink-0 text-stone" />
                  <span className="flex min-w-0 grow flex-col">
                    <span className="text-[16px] font-semibold text-ink">Start from last week</span>
                    <span className="text-support text-stone">
                      {previous.points.length === 1
                        ? "Brings its one prayer over. Change or remove it as you like."
                        : `Brings all ${previous.points.length} over. Change or remove any of them.`}
                    </span>
                  </span>
                </button>
              ) : null}
              <ol className="m-0 mt-6 list-none border-t border-line p-0" {...drag.listProps}>
                {items.map((p, i) => {
                  // Once the partner has prayed it (on any day this week), it can be reordered
                  // but not edited or removed.
                  const locked = w.locked.includes(p.id);
                  const body = (
                    <>
                      <span className="text-bodylg font-semibold tracking-[-0.01em]">
                        {p.title || "Untitled prayer"}
                      </span>
                      {p.text ? <span className="text-support text-stone">{p.text}</span> : null}
                      <span className="text-support text-stone">{daysLabel(p.weekdays)}</span>
                      {locked ? (
                        <span className="text-support text-stone">
                          {partner} has prayed this one.
                        </span>
                      ) : null}
                    </>
                  );
                  return (
                    <li
                      key={p.id}
                      // Held row lifts onto its own layer and tracks the drag offset via transform.
                      className={cx(
                        "flex min-h-[72px] items-center gap-3 border-b border-line",
                        drag.held?.index === i
                          ? "relative z-10 rounded-btn border-transparent bg-surface"
                          : "transition-transform duration-150",
                      )}
                      style={
                        drag.held?.index === i
                          ? { transform: `translateY(${drag.held.offset}px)` }
                          : undefined
                      }
                    >
                      <span className="tabular w-[22px] self-start pt-[17px] text-[13px] font-semibold text-stone">
                        {String(i + 1).padStart(2, "0")}
                      </span>
                      {locked ? (
                        <div className="flex min-h-[72px] grow flex-col justify-center gap-0.5 py-3 text-ink">
                          {body}
                        </div>
                      ) : (
                        <Link
                          href={routes.prayerEdit(p.id)}
                          className="press flex min-h-[72px] grow flex-col justify-center gap-0.5 text-ink no-underline"
                        >
                          {body}
                        </Link>
                      )}
                      <button
                        type="button"
                        aria-label={`Reorder ${p.title}. Use the arrow keys to move it.`}
                        {...drag.handleProps(i)}
                        onKeyDown={(e) => {
                          if (e.key === "ArrowUp") {
                            e.preventDefault();
                            drag.moveByKey(i, i - 1);
                          }
                          if (e.key === "ArrowDown") {
                            e.preventDefault();
                            drag.moveByKey(i, i + 1);
                          }
                        }}
                        className="press -mr-3 flex h-11 w-11 touch-none cursor-grab items-center justify-center text-stone active:cursor-grabbing"
                      >
                        <Icon name="grip" size={22} strokeWidth={2.2} />
                      </button>
                    </li>
                  );
                })}
                {items.length < MAX ? (
                  <li>
                    <Link
                      href={routes.prayerEdit()}
                      className="press flex h-[60px] items-center gap-3 text-[16px] font-semibold text-plum no-underline"
                    >
                      <span className="flex w-[22px]">
                        <Icon name="plus" size={20} strokeWidth={1.8} />
                      </span>
                      Add a prayer
                    </Link>
                  </li>
                ) : (
                  <li className="py-4 text-support text-stone">
                    That&rsquo;s the ten for this week.
                  </li>
                )}
              </ol>
            </Main>
            <div className="flex shrink-0 flex-col gap-2 px-6 pt-4 pb-safe">
              {shared ? (
                <LinkButton href={routes.home}>Done</LinkButton>
              ) : (
                <>
                  <Button onClick={() => setConfirm(true)} disabled={items.length === 0 || !saved}>
                    Publish prayers
                  </Button>
                  <LinkButton href={routes.home} variant="text">
                    Save draft and finish later
                  </LinkButton>
                </>
              )}
            </div>

            <Sheet
              open={confirm}
              onClose={() => setConfirm(false)}
              title="Ready to share these prayers?"
              labelledBy="pub-h"
            >
              <Para>
                {/* "they" is used since a display name doesn't indicate pronoun. */}
                {partner} will see {items.length === 1 ? "it" : `all ${items.length}`}. You can keep
                adding and changing them all week &mdash; only the ones they&rsquo;ve already prayed
                stay as they are.
              </Para>
              <ol className="m-0 list-none border-y border-line py-1">
                {items.map((p, i) => (
                  <li key={p.id} className="flex h-10 items-center gap-3.5 text-[16px] font-medium">
                    <span className="tabular w-[22px] text-[13px] font-semibold text-stone">
                      {String(i + 1).padStart(2, "0")}
                    </span>
                    {p.title}
                  </li>
                ))}
              </ol>
              <div className="flex flex-col gap-1 pt-1">
                <Button onClick={doPublish} loading={publish.isPending}>
                  {publish.isPending ? "Sharing" : `Share with ${partner}`}
                </Button>
                <Button variant="text" onClick={() => setConfirm(false)}>
                  Keep editing
                </Button>
              </div>
            </Sheet>
          </>
        );
      }}
    </QueryState>
  );
}
