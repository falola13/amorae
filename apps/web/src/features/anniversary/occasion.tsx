"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { AnimatedMark } from "@/components/icons";
import { useCouple } from "@/features/couple/hooks";
import { iso, longDateYear } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";
import { count, daysTogether, isTheDay, markOpeningSeen, openingSeen, yearsWord } from "./derive";
import { useAnniversary } from "./hooks";

// How many mounted components want the colours on: the shell on the day, and
// Our day's preview. Counted so one leaving doesn't turn them off for the other.
let holders = 0;

/** Gives the whole app the anniversary's colours while `active` (globals.css). */
export function useOccasionTheme(active: boolean) {
  useEffect(() => {
    if (!active) return;
    const root = document.documentElement;
    holders += 1;
    root.dataset.occasion = "anniversary";
    return () => {
      holders -= 1;
      if (holders === 0) delete root.dataset.occasion;
    };
  }, [active]);
}

/**
 * Mounted once in the app shell. On the day itself it turns the app's
 * colours over, and the first time the day is opened on this device it
 * plays the opening. Every other day it renders nothing and costs one
 * cached request.
 */
export function Occasion() {
  const anniversary = useAnniversary();
  const a = anniversary.data;
  const theDay = isTheDay(a);
  useOccasionTheme(theDay);

  // Which day was closed here, so closing re-renders; openingSeen covers
  // every later visit. Never rendered on the server: the shell shows the
  // splash until the couple has loaded, so localStorage is there to read.
  const [closed, setClosed] = useState<string | null>(null);
  if (!theDay || !a || closed === a.on || openingSeen(a.on)) return null;
  return (
    <Opening
      on={a.on}
      years={a.years}
      onClose={() => {
        markOpeningSeen(a.on);
        setClosed(a.on);
      }}
    />
  );
}

// Fixed, not random: the same drift on every render and on the server.
const LIGHTS = [
  { left: 6, size: 4, delay: 0, duration: 11 },
  { left: 14, size: 6, delay: 3.2, duration: 13 },
  { left: 23, size: 3, delay: 6.1, duration: 9 },
  { left: 31, size: 5, delay: 1.4, duration: 12 },
  { left: 40, size: 3, delay: 8.3, duration: 10 },
  { left: 48, size: 7, delay: 4.6, duration: 14 },
  { left: 57, size: 4, delay: 0.8, duration: 11 },
  { left: 65, size: 3, delay: 5.5, duration: 9.5 },
  { left: 72, size: 6, delay: 2.3, duration: 12.5 },
  { left: 80, size: 4, delay: 7.4, duration: 10.5 },
  { left: 88, size: 5, delay: 3.9, duration: 13.5 },
  { left: 95, size: 3, delay: 6.8, duration: 9 },
];

/**
 * The opening: deep wine, slow rising light, the mark drawing itself, then
 * the years. Its colours are its own rather than the tokens — it is the
 * same evening in light mode and dark.
 */
export function Opening({
  on,
  years,
  onClose,
}: {
  on: string;
  years: number;
  onClose: () => void;
}) {
  const router = useRouter();
  const couple = useCouple();
  const primary = useRef<HTMLButtonElement>(null);
  const me = couple.data?.me.display_name ?? "";
  const partner = couple.data?.partner?.display_name;
  const started = couple.data?.started_on;
  const days = started ? daysTogether(started, iso(today())) : null;

  useEffect(() => {
    const timer = setTimeout(() => primary.current?.focus(), 1900);
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      clearTimeout(timer);
      window.removeEventListener("keydown", onKey);
      document.body.style.overflow = overflow;
    };
  }, [onClose]);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="occasion-h"
      className="fixed inset-0 z-50 flex animate-fade flex-col overflow-hidden text-[#fbeef2]"
      style={{
        background:
          "radial-gradient(120% 70% at 50% 0%, #8a3a60 0%, transparent 60%), linear-gradient(170deg, #3d1428 0%, #5a2246 55%, #2b0f1f 100%)",
      }}
    >
      <div aria-hidden="true" className="pointer-events-none absolute inset-0">
        {LIGHTS.map((l) => (
          <span
            key={l.left}
            className="absolute bottom-[-12px] animate-drift rounded-full opacity-0"
            style={{
              left: `${l.left}%`,
              width: l.size,
              height: l.size,
              background:
                "radial-gradient(circle, #ffe3ec 0%, rgba(255,214,228,0.4) 60%, transparent 100%)",
              boxShadow: "0 0 12px rgba(255, 200, 220, 0.6)",
              animationDelay: `${l.delay}s`,
              animationDuration: `${l.duration}s`,
            }}
          />
        ))}
      </div>

      <div
        className="relative mx-auto flex w-full max-w-[520px] grow flex-col items-center justify-center px-8 text-center"
        style={{ paddingTop: "calc(var(--safe-top) + 24px)" }}
      >
        <AnimatedMark size={64} className="text-[#f6c7d8]" />
        <div className="mt-7 animate-occasion-1 text-micro uppercase tracking-[0.16em] text-[#e9c886]">
          {longDateYear(on)}
        </div>
        <h1
          id="occasion-h"
          className="m-0 mt-3 animate-occasion-2 text-[52px] font-semibold leading-[1.05] tracking-[-0.035em]"
        >
          {yearsWord(years)}
        </h1>
        <p className="m-0 mt-4 animate-occasion-3 text-[19px] leading-[1.5] text-[#f5dbe4]">
          Happy anniversary{me ? `, ${me}` : ""}
          {partner ? ` & ${partner}` : ""}.
        </p>
        {days ? (
          <p className="m-0 mt-2 animate-occasion-3 text-[15px] leading-[1.5] text-[#dcb6c5]">
            {count(days)} days of choosing each other.
          </p>
        ) : null}
      </div>

      <div className="relative mx-auto flex w-full max-w-[520px] animate-occasion-4 flex-col gap-1 px-6 pb-safe">
        <button
          ref={primary}
          type="button"
          onClick={() => {
            onClose();
            router.push(routes.ourDay());
          }}
          className="press h-[54px] w-full rounded-btn bg-[#fbeef2] text-[16px] font-semibold text-[#4a1a35]"
        >
          Open our day
        </button>
        <button
          type="button"
          onClick={onClose}
          className="press h-11 w-full text-[15px] font-semibold text-[#f0cbd8]"
        >
          Not now
        </button>
      </div>
    </div>
  );
}
