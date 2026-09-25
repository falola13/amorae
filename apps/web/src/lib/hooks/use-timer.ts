"use client";

import { useEffect, useState } from "react";

import { useUI } from "@/lib/store/ui";

/** Stopwatch backed by a shared start timestamp (lib/store/ui.ts), so it keeps
 *  running across screens. `toggle` starts it from zero or stops it. */
export function useTimer() {
  const startedAt = useUI((s) => s.timerStartedAt);
  const toggle = useUI((s) => s.toggleTimer);
  const [secs, setSecs] = useState<number | null>(null);
  useEffect(() => {
    if (startedAt === null) {
      const t = setTimeout(() => setSecs(null), 0);
      return () => clearTimeout(t);
    }
    const update = () => setSecs(Math.floor((Date.now() - startedAt) / 1000));
    const t = setInterval(update, 1000);
    const first = setTimeout(update, 0);
    return () => {
      clearInterval(t);
      clearTimeout(first);
    };
  }, [startedAt]);
  const label =
    secs === null ? null : `${Math.floor(secs / 60)}:${String(secs % 60).padStart(2, "0")}`;
  return { running: startedAt !== null, label, toggle };
}
