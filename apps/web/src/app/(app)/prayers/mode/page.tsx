"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, type ReactNode } from "react";

import { LinkButton, Skeleton } from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { PrayerSession } from "@/features/prayers/components/prayer-session";
import { QuietPrayer } from "@/features/prayers/components/quiet-prayer";
import { startIndex } from "@/features/prayers/derive";
import { useWeek } from "@/features/prayers/hooks";
import { useTimer } from "@/lib/hooks/use-timer";
import { routes } from "@/lib/routes";
import { useUI } from "@/lib/store/ui";

export default function PrayerModePage() {
  return (
    <Suspense fallback={null}>
      <PrayerMode />
    </Suspense>
  );
}

function PrayerMode() {
  const week = useWeek();
  const params = useSearchParams();
  const quiet = params.get("quiet") === "1";
  const i = useUI((s) => s.prayerIndex);
  const setI = useUI((s) => s.setPrayerIndex);
  const timer = useTimer();

  // ?at= is applied once per value (not on every `i` change), or Next/Previous would snap back to it.
  const appliedAt = useRef<string | null>(null);
  useEffect(() => {
    const w = week.data;
    if (!w) return;
    const at = params.get("at");
    if (at !== null && appliedAt.current !== at) {
      appliedAt.current = at;
      setI(startIndex(w, at));
    } else if (i === null || i >= w.points.length) {
      setI(startIndex(w, null));
    }
  }, [week.data, params, i, setI]);

  if (quiet) return <QuietPrayer timer={timer} />;

  const loading = (
    <div className="mx-auto flex min-h-dvh w-full max-w-[520px] flex-col grain bg-paper px-8 pt-24">
      <Skeleton />
    </div>
  );
  // No tab bar in prayer mode, so the notice needs its own back link; bg-bg lifts its card off the paper background.
  const frame = (notice: ReactNode) => (
    <div className="mx-auto flex min-h-dvh w-full max-w-[520px] flex-col gap-2 grain bg-paper px-6 pt-24 [&>[role]]:bg-bg">
      {notice}
      <LinkButton href={routes.prayers} variant="text">
        Back to prayers
      </LinkButton>
    </div>
  );
  return (
    <QueryState queries={[week]} loading={loading} frame={frame}>
      {(w) => {
        if (w.points.length === 0) return <QuietPrayer timer={timer} />;
        if (i === null || i >= w.points.length) return loading;
        return <PrayerSession week={w} index={i} onIndexChange={setI} timer={timer} />;
      }}
    </QueryState>
  );
}
