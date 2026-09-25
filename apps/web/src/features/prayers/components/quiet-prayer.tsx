"use client";

import Link from "next/link";

import { Icon } from "@/components/icons";
import { Button } from "@/components/ui/kit";
import type { useTimer } from "@/lib/hooks/use-timer";
import { routes } from "@/lib/routes";

/** No list today — just space and an optional timer. Shown for quiet prayer (?quiet=1) or an empty week. */
export function QuietPrayer({ timer }: { timer: ReturnType<typeof useTimer> }) {
  return (
    <div className="mx-auto flex min-h-dvh w-full max-w-[520px] flex-col bg-paper">
      <div className="shrink-0" style={{ height: "calc(var(--safe-top) + 12px)" }} />
      <div className="flex h-11 shrink-0 items-center px-3">
        <Link
          href={routes.home}
          aria-label="Leave quiet prayer"
          className="press flex h-11 w-11 items-center justify-center text-ink"
        >
          <Icon name="x" size={22} />
        </Link>
      </div>
      <main className="flex grow animate-page flex-col justify-center gap-5 px-8 pb-6">
        <Icon name="moon" size={28} strokeWidth={1.4} className="text-plum" />
        <h1 className="m-0 text-display">Take a quiet moment.</h1>
        <p className="m-0 text-reading text-stone">
          No list today. Just you, and whatever is on your heart.
        </p>
        {timer.label ? (
          <div
            aria-live="polite"
            className="tabular text-[40px] font-medium tracking-[-0.03em] text-plum"
          >
            {timer.label}
          </div>
        ) : null}
      </main>
      <div className="flex shrink-0 flex-col gap-1 px-6 pb-safe">
        <Button
          variant={timer.running ? "secondary" : "primary"}
          icon="clock"
          onClick={timer.toggle}
        >
          {timer.running ? "Stop" : "Start a quiet timer"}
        </Button>
      </div>
    </div>
  );
}
