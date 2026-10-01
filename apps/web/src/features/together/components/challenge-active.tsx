"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { Main } from "@/components/layout/screen";
import {
  BottomActions,
  Button,
  Micro,
  Para,
  Sheet,
  StatusMark,
  Title,
  cx,
} from "@/components/ui/kit";
import { useChallengeDay, useLeaveChallenge } from "@/features/together/hooks";
import type { Challenge, ChallengeDay } from "@/lib/api/types";
import { iso } from "@/lib/dates";
import { keys } from "@/lib/query/keys";
import { today } from "@/lib/today";
import {
  cheerOn,
  dayDateLabel,
  isCatchUp,
  isMarkable,
  isScheduled,
  opensLabel,
  ownerLabel,
  progressLabel,
} from "../challenges";
import { ChallengeNoteSheet } from "./challenge-note-sheet";

const chip = "press h-9 rounded-full px-3.5 text-[14px] font-semibold";

function DayItem({
  d,
  c,
  partner,
  todayIso,
  markable,
  onDone,
  onSkip,
  onNote,
}: {
  d: ChallengeDay;
  c: Challenge;
  partner: string;
  todayIso: string;
  /** False on a partner's own challenge: you read it, you do not mark it. */
  markable: boolean;
  onDone: () => void;
  onSkip: () => void;
  onNote: () => void;
}) {
  const isToday = d.n === c.today_n;
  const catchUp = isCatchUp(d, c.today_n);
  const marked = d.done || d.skipped;
  return (
    <li className={cx("flex flex-col gap-2 border-b border-line py-3.5", !d.open && "opacity-55")}>
      <div className="flex items-start gap-3.5">
        <span className="pt-0.5">
          {d.done ? (
            <StatusMark done label="Done" />
          ) : (
            <span
              aria-hidden="true"
              className={cx(
                "box-border block h-[22px] w-[22px] shrink-0 rounded-full",
                isToday && d.open ? "border-2 border-plum" : "border-[1.5px] border-edge",
              )}
            />
          )}
        </span>
        <span className="flex min-w-0 grow flex-col">
          <span className="text-[12px] font-semibold text-stone">
            Day {d.n} · {dayDateLabel(d.date)}
            {d.skipped ? " · skipped" : ""}
            {catchUp ? " · catch up" : ""}
          </span>
          <span
            className={cx(
              "text-[16px]",
              isToday && d.open ? "font-semibold text-ink" : "font-medium text-stone",
            )}
          >
            {d.text}
          </span>
          {!d.open ? (
            <span className="mt-0.5 text-[13px] text-stone">{opensLabel(d.date, todayIso)}</span>
          ) : null}
        </span>
        {isToday && d.open ? <span className="text-[12px] font-bold text-plum">Today</span> : null}
      </div>

      {/* Label only, never a control: you can see the partner's status but not change it (DEC-30). */}
      {d.partner_done || d.partner_skipped ? (
        <span className="pl-9 text-[12px] font-semibold text-stone">
          {partner} {d.partner_done ? "did" : "skipped"}
        </span>
      ) : null}

      {d.note ? (
        <p className="m-0 pl-9 text-[15px] leading-snug text-ink">
          <span className="font-semibold text-stone">You: </span>
          {d.note}
        </p>
      ) : null}
      {d.partner_note ? (
        <p className="m-0 pl-9 text-[15px] leading-snug text-ink">
          <span className="font-semibold text-stone">{partner}: </span>
          {d.partner_note}
        </p>
      ) : null}

      {d.open && markable ? (
        <div className="flex flex-wrap gap-1 pl-9">
          {!d.done ? (
            <button type="button" onClick={onDone} className={cx(chip, "bg-plum-tint text-plum")}>
              {catchUp ? "Catch up" : "Done"}
            </button>
          ) : (
            <button type="button" onClick={onNote} className={cx(chip, "text-stone")}>
              {d.note ? "Edit note" : "Add a note"}
            </button>
          )}
          {!marked ? (
            <button type="button" onClick={onSkip} className={cx(chip, "text-stone")}>
              Skip
            </button>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}

/** A challenge that is running: one shared "Day N of M", each day's marks and notes. */
export function ChallengeActive({ c, partner }: { c: Challenge; partner: string }) {
  const qc = useQueryClient();
  const set = useChallengeDay();
  const leave = useLeaveChallenge();
  const [ending, setEnding] = useState(false);
  // Which day's note sheet is open, and whether that is the moment of marking it done.
  const [noting, setNoting] = useState<{ n: number; marking: boolean } | null>(null);
  const todayIso = iso(today());
  const todayDay = c.days.find((d) => d.n === c.today_n && d.open);
  const markable = isMarkable(c);
  const scheduled = isScheduled(c);
  const owner = ownerLabel(c, partner);
  const todayOpen = markable && todayDay && !todayDay.done && !todayDay.skipped ? todayDay : null;
  const noted = noting ? c.days.find((d) => d.n === noting.n) : undefined;

  const skip = (n: number) => set.mutate({ id: c.id, n, patch: { skipped: true } });
  const save = (note: string) => {
    if (!noting) return;
    const { n, marking } = noting;
    const close = () => setNoting(null);
    // Marking and noting go in one request, so the last day's note is not lost to the challenge closing.
    const patch = marking ? { done: true, ...(note ? { note } : {}) } : { note };
    set.mutate(
      { id: c.id, n, patch },
      {
        onSuccess: (v) => {
          close();
          // The last day closes it: show the finish view here, right away.
          if (v.status !== "active") qc.setQueryData(keys.challengeById(v.id), v);
        },
        onQueued: close,
      },
    );
  };
  const end = () =>
    // Stay on this page: once it refreshes as ended, the finish view takes its place.
    leave.mutate(c.id, {
      onSuccess: () => setEnding(false),
      onQueued: () => setEnding(false),
    });

  return (
    <>
      <Main>
        <div className="pt-2">
          <Micro>
            {progressLabel(c)}
            {owner ? ` · ${owner}` : ""}
          </Micro>
        </div>
        <Title className="mt-2">{c.title}</Title>
        <Para className="mt-1.5">
          {!markable
            ? cheerOn(partner)
            : scheduled
              ? "Nothing to do yet. Day 1 opens when it starts."
              : "One small thing a day. Skip any day you need to."}
        </Para>
        <ol className="m-0 mt-4 list-none border-t border-line p-0">
          {c.days.map((d) => (
            <DayItem
              key={d.n}
              d={d}
              c={c}
              partner={partner}
              todayIso={todayIso}
              markable={markable}
              onDone={() => setNoting({ n: d.n, marking: true })}
              onSkip={() => skip(d.n)}
              onNote={() => setNoting({ n: d.n, marking: false })}
            />
          ))}
        </ol>
      </Main>
      {markable ? (
        <BottomActions>
          {todayOpen ? (
            <>
              <Button icon="check" onClick={() => setNoting({ n: todayOpen.n, marking: true })}>
                I did today&rsquo;s
              </Button>
              <Button variant="text" onClick={() => skip(todayOpen.n)}>
                Skip today
              </Button>
            </>
          ) : null}
          <Button variant="text" className="text-stone" onClick={() => setEnding(true)}>
            End this challenge
          </Button>
        </BottomActions>
      ) : null}

      <ChallengeNoteSheet
        key={noting ? `${noting.n}-${noting.marking}` : "closed"}
        open={noting !== null}
        onClose={() => setNoting(null)}
        partner={partner}
        initial={noting?.marking ? "" : noted?.note}
        marking={noting?.marking ?? true}
        busy={set.isPending}
        onSave={save}
      />
      <Sheet
        open={ending}
        onClose={() => setEnding(false)}
        title="End this challenge?"
        labelledBy="end-ch-h"
      >
        <Para>
          {c.kind === "mine"
            ? "It ends and stays in your story as ended early."
            : "It ends for both of you and stays in your story as ended early."}
        </Para>
        <div className="flex flex-col gap-1">
          <Button
            variant="secondary"
            className="border-red text-red"
            loading={leave.isPending}
            onClick={end}
          >
            End it
          </Button>
          <Button variant="text" onClick={() => setEnding(false)}>
            Keep going
          </Button>
        </div>
      </Sheet>
    </>
  );
}
