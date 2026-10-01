"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Main } from "@/components/layout/screen";
import { BareTextarea, Button, Micro, Para, Title, cx } from "@/components/ui/kit";
import {
  useActiveChallenges,
  useChallengeReflection,
  useChallengeTemplates,
  usePastChallenges,
} from "@/features/together/hooks";
import type { Challenge, ChallengeDay } from "@/lib/api/types";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";
import { dayDateLabel, doneCount, overLabel, pickSuggestions, runningKeys } from "../challenges";
import { StartChallengeSheet } from "./start-challenge-sheet";

function Side({
  who,
  done,
  skipped,
  note,
}: {
  who: string;
  done?: boolean;
  skipped?: boolean;
  note?: string;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="text-[12px] font-semibold text-stone">{who}</span>
      <span className={cx("text-[14px]", done ? "font-semibold text-ink" : "text-stone")}>
        {done ? "Did it" : skipped ? "Skipped" : "Not marked"}
      </span>
      {note ? <span className="break-words text-[15px] leading-snug text-ink">{note}</span> : null}
    </div>
  );
}

function DayRow({ d, partner }: { d: ChallengeDay; partner: string }) {
  return (
    <li className="flex flex-col gap-2 border-b border-line py-3.5">
      <span className="flex flex-col">
        <span className="text-[12px] font-semibold text-stone">
          Day {d.n} · {dayDateLabel(d.date)}
        </span>
        <span className="text-[16px] font-medium text-ink">{d.text}</span>
      </span>
      <div className="grid grid-cols-2 gap-4">
        <Side who="You" done={d.done} skipped={d.skipped} note={d.note} />
        <Side
          who={partner}
          done={d.partner_done}
          skipped={d.partner_skipped}
          note={d.partner_note}
        />
      </div>
    </li>
  );
}

/** The story of a challenge that is over: both people's days and notes, a place to look back,
 *  and, if nothing else is running, where to go next. */
export function ChallengeFinish({
  c,
  partner,
  canStartAnother,
}: {
  c: Challenge;
  partner: string;
  canStartAnother: boolean;
}) {
  const router = useRouter();
  const reflect = useChallengeReflection();
  const templates = useChallengeTemplates();
  const past = usePastChallenges();
  const active = useActiveChallenges();
  const [text, setText] = useState(c.reflection ?? "");
  const [starting, setStarting] = useState<string | null>(null);
  const counts = doneCount(c);
  const changed = text.trim() !== (c.reflection ?? "").trim();

  const suggestions = canStartAnother
    ? pickSuggestions(
        (templates.data ?? []).filter((t) => !runningKeys(active.data).has(t.key)),
        [...(past.data ?? []).map((p) => p.template), c.template],
        today(),
      )
    : [];
  const picked = suggestions.find((t) => t.key === starting) ?? null;

  return (
    <Main>
      <div className="pt-2">
        <Micro>{overLabel(c.status)}</Micro>
      </div>
      <Title className="mt-2">{c.title}</Title>
      <Para className="mt-1.5">
        {c.status === "ended"
          ? `It ended early. You marked ${counts.mine} of ${c.days.length}, ${partner} ${counts.partner}.`
          : `${c.days.length} days. You marked ${counts.mine}, ${partner} ${counts.partner}.`}
      </Para>

      <ol className="m-0 mt-4 list-none border-t border-line p-0">
        {c.days.map((d) => (
          <DayRow key={d.n} d={d} partner={partner} />
        ))}
      </ol>

      <section className="mt-6 flex flex-col gap-2">
        <BareTextarea
          label="Looking back"
          rows={3}
          maxLength={1000}
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="What stayed with you?"
          className="min-h-[84px] text-[17px] leading-[1.6]"
        />
        {changed ? (
          <div>
            <Button
              variant="secondary"
              loading={reflect.isPending}
              onClick={() => reflect.mutate({ id: c.id, text: text.trim() })}
            >
              Save
            </Button>
          </div>
        ) : null}
        {c.partner_reflection ? (
          <div className="mt-3 flex flex-col gap-1 rounded-card border border-line bg-surface px-4 py-3.5">
            <Micro>{partner} looking back</Micro>
            <Para>{c.partner_reflection}</Para>
          </div>
        ) : null}
      </section>

      {suggestions.length > 0 ? (
        <section className="mt-8 flex flex-col pb-6">
          <Micro>Start another</Micro>
          <ol className="m-0 mt-1 list-none border-t border-line p-0">
            {suggestions.map((t) => (
              <li key={t.key} className="border-b border-line">
                <button
                  type="button"
                  onClick={() => setStarting(t.key)}
                  className="press flex min-h-[64px] w-full items-center gap-3.5 py-3 text-left"
                >
                  <span className="flex grow flex-col gap-0.5">
                    <span className="text-[16px] font-semibold text-ink">{t.title}</span>
                    <span className="text-support text-stone">{t.blurb}</span>
                  </span>
                  <span className="shrink-0 text-[13px] font-semibold text-stone">
                    {t.days} days
                  </span>
                </button>
              </li>
            ))}
          </ol>
        </section>
      ) : null}
      <StartChallengeSheet
        template={picked}
        onClose={() => setStarting(null)}
        onStarted={() => router.replace(routes.challenges)}
      />
    </Main>
  );
}
