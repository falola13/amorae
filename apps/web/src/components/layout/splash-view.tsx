"use client";

import { useEffect, useState } from "react";

import { AnimatedMark } from "@/components/icons";

/** After this long, say so, rather than breathing at someone indefinitely. */
const SLOW_MS = 8000;

/** Opening screen while the couple loads. The waiting indicator fades in at
 *  1.2s, so it only shows on slow loads; prefers-reduced-motion just shows
 *  the finished composition (tokens in globals.css). */
export function SplashView({ label = "Opening your space" }: { label?: string }) {
  const [slow, setSlow] = useState(false);
  useEffect(() => {
    const timer = setTimeout(() => setSlow(true), SLOW_MS);
    return () => clearTimeout(timer);
  }, []);

  return (
    <div className="mx-auto flex min-h-[100dvh] w-full max-w-[520px] flex-col bg-bg md:max-w-[640px]">
      <div className="flex grow flex-col items-center justify-center gap-3.5 pb-10">
        <AnimatedMark size={56} className="text-plum md:h-16 md:w-16" />
        <div className="animate-splash-word text-[30px] font-semibold tracking-[-0.03em] md:text-display">
          Amorae
        </div>
        <div className="animate-splash-tag text-support text-stone">Two hearts, one faith.</div>
      </div>
      <div
        role="status"
        className="flex shrink-0 animate-splash-wait flex-col items-center gap-3.5 pb-[74px]"
      >
        <div className="h-0.5 w-10 animate-breathe rounded-full bg-plum" />
        <div className="text-[13px] text-stone">
          {slow ? "Still opening. This is taking longer than usual." : label}
        </div>
      </div>
    </div>
  );
}
