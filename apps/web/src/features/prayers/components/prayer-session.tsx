"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { Icon } from "@/components/icons";
import { Button, cx } from "@/components/ui/kit";
import { Scripture } from "@/components/ui/scripture";
import { useSetCompleted } from "@/features/prayers/hooks";
import type { PrayerWeek } from "@/lib/api/types";
import type { useTimer } from "@/lib/hooks/use-timer";
import { routes } from "@/lib/routes";

/** One prayer at a time: title, text, scripture, a "prayed" toggle, and next/previous. */
export function PrayerSession({
  week: w,
  index: i,
  onIndexChange: setI,
  timer,
}: {
  week: PrayerWeek;
  index: number;
  onIndexChange: (i: number) => void;
  timer: ReturnType<typeof useTimer>;
}) {
  const complete = useSetCompleted();
  const router = useRouter();
  const p = w.points[i];
  const n = w.points.length;
  const prayed = w.my_completed.includes(p.id);
  const allDone = w.my_completed.length === n;
  const last = i === n - 1;

  return (
    <div className="mx-auto flex min-h-dvh w-full max-w-[520px] flex-col grain bg-paper md:max-w-[640px]">
      <div className="shrink-0" style={{ height: "calc(var(--safe-top) + 12px)" }} />
      <div className="flex h-11 shrink-0 items-center justify-between px-3">
        <Link
          href={routes.prayers}
          aria-label="Leave prayer mode"
          className="press flex h-11 w-11 items-center justify-center text-ink"
        >
          <Icon name="x" size={22} />
        </Link>
        {/* Both colors are ≥3:1 contrast (WCAG 1.4.11); current point is also marked by shape, not color alone. */}
        <div aria-hidden="true" className="flex w-32 items-center gap-1">
          {w.points.map((x, k) => (
            <div
              key={x.id}
              className={cx(
                "grow basis-0 rounded-full",
                k === i ? "h-2" : "h-1",
                w.my_completed.includes(x.id) ? "bg-plum" : "bg-edge",
              )}
            />
          ))}
        </div>
        <button
          type="button"
          aria-label={timer.running ? "Stop the timer" : "Start a quiet timer"}
          aria-pressed={timer.running}
          onClick={timer.toggle}
          className={cx(
            "press flex h-11 min-w-11 items-center justify-center gap-1 px-1 text-[13px] font-semibold",
            timer.running ? "tabular text-plum" : "text-stone",
          )}
        >
          <Icon name="clock" size={22} />
          {timer.label}
        </button>
      </div>

      <main key={p.id} className="flex grow animate-page flex-col justify-center gap-4 px-8 pb-4">
        <div aria-live="polite" className="tabular text-micro uppercase text-stone">
          Prayer {i + 1} of {n}
        </div>
        <h1
          className="m-0 -mt-1 text-[30px] font-semibold leading-[1.18] tracking-[-0.025em] md:text-display"
          data-selectable
        >
          {p.title}
        </h1>
        {p.text ? (
          // From `md`, body uses the larger reading size — further from the eye on tablet/laptop.
          <p className="m-0 text-[19px] leading-[1.6] md:text-reading" data-selectable>
            {p.text}
          </p>
        ) : null}
        {p.scripture ? <Scripture reference={p.scripture} verse={p.verse} faint /> : null}
      </main>

      <div className="flex shrink-0 flex-col gap-1.5 px-6 pb-safe">
        {prayed ? (
          <div
            role="status"
            className="flex h-[54px] animate-rise items-center justify-between rounded-btn bg-green-tint pl-[18px] pr-1.5"
          >
            <div className="flex items-center gap-2.5 text-[16px] font-semibold text-green">
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                aria-hidden="true"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M5 12.5l4.5 4.5L19 7.5" className="draw-check animate-draw" />
              </svg>
              Prayed
            </div>
            <button
              type="button"
              onClick={() => complete.mutate({ pointId: p.id, done: false })}
              className="press h-11 px-3 text-[15px] font-semibold text-green"
            >
              Undo
            </button>
          </div>
        ) : (
          <Button icon="check" onClick={() => complete.mutate({ pointId: p.id, done: true })}>
            I&rsquo;ve prayed
          </Button>
        )}
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => setI(Math.max(0, i - 1))}
            disabled={i === 0}
            className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-ink disabled:opacity-35"
          >
            <Icon name="left" size={20} />
            Previous
          </button>
          {allDone ? (
            <button
              type="button"
              onClick={() => router.push(routes.prayersDone)}
              className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-plum"
            >
              Finish
              <Icon name="right" size={20} />
            </button>
          ) : !last ? (
            <button
              type="button"
              onClick={() => setI(Math.min(n - 1, i + 1))}
              className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-ink"
            >
              Next
              <Icon name="right" size={20} />
            </button>
          ) : (
            <Link
              href={routes.prayers}
              className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-ink no-underline"
            >
              Done for now
              <Icon name="right" size={20} />
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}
