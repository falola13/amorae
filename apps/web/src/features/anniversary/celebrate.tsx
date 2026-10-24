"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

import { yearsWord } from "./derive";

/**
 * Puts a full-screen layer on <body>. Screens animate in with a transform
 * (animate-page), and a transformed ancestor would make `position: fixed`
 * mean "fixed to that screen" instead of to the viewport.
 */
export function OnTop({ children }: { children: ReactNode }) {
  const [mounted, setMounted] = useState(false);
  useEffect(() => {
    const t = setTimeout(() => setMounted(true), 0);
    return () => clearTimeout(t);
  }, []);
  return mounted ? createPortal(children, document.body) : null;
}

/* ---------- Confetti ---------- */

const COLORS = ["#e9c77f", "#c99a4b", "#fff1c9", "#f2a7bf", "#d96a8f", "#fbeef2", "#b5577f"];

interface Bit {
  x: number;
  y: number;
  vx: number;
  vy: number;
  rot: number;
  vr: number;
  size: number;
  color: string;
  /** 0 a flake of gold leaf, 1 a petal, 2 a small heart. */
  shape: 0 | 1 | 2;
  wobble: number;
  born: number;
}

const reducedMotion = () =>
  typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

function heart(ctx: CanvasRenderingContext2D, s: number) {
  ctx.beginPath();
  ctx.moveTo(0, s * 0.35);
  ctx.bezierCurveTo(-s, -s * 0.35, -s * 0.45, -s, 0, -s * 0.45);
  ctx.bezierCurveTo(s * 0.45, -s, s, -s * 0.35, 0, s * 0.35);
  ctx.fill();
}

/**
 * Gold leaf, petals and the odd heart, thrown up from a point and left to
 * drift down. Every change of `fire` throws another handful; 0 throws none.
 * Drawn on one canvas and stopped the moment the last piece has fallen, so
 * it costs nothing at rest. Reduced motion: no confetti at all.
 */
export function Confetti({ fire, originY = 0.38 }: { fire: number; originY?: number }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const bits = useRef<Bit[]>([]);
  const frame = useRef<number | null>(null);

  useEffect(() => {
    if (!fire || reducedMotion()) return;
    const el = canvas.current;
    const ctx = el?.getContext("2d");
    if (!el || !ctx) return;

    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    const w = window.innerWidth;
    const h = window.innerHeight;
    el.width = w * dpr;
    el.height = h * dpr;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    const now = performance.now();
    const rand = (a: number, b: number) => a + Math.random() * (b - a);
    const add = (x: number, y: number, speed: number, spread: number, n: number, delay: number) => {
      for (let i = 0; i < n; i++) {
        const angle = -Math.PI / 2 + rand(-spread, spread);
        const v = rand(speed * 0.45, speed);
        const shape = (Math.random() < 0.55 ? 0 : Math.random() < 0.7 ? 1 : 2) as Bit["shape"];
        bits.current.push({
          x,
          y,
          vx: Math.cos(angle) * v,
          vy: Math.sin(angle) * v,
          rot: rand(0, Math.PI * 2),
          vr: rand(-0.25, 0.25),
          size: shape === 2 ? rand(5, 8) : rand(4, 9),
          color: COLORS[Math.floor(Math.random() * COLORS.length)],
          shape,
          wobble: rand(0, Math.PI * 2),
          born: now + delay,
        });
      }
    };
    // One burst from the centre, then two from the lower corners a beat later.
    add(w / 2, h * originY, 15, 1.25, 110, 0);
    add(w * 0.08, h * 0.95, 17, 0.35, 45, 180);
    add(w * 0.92, h * 0.95, 17, 0.35, 45, 180);

    const tick = (t: number) => {
      ctx.clearRect(0, 0, w, h);
      bits.current = bits.current.filter((b) => b.y < h + 40);
      for (const b of bits.current) {
        if (t < b.born) continue;
        b.vy += 0.32;
        b.vx *= 0.985;
        b.vy *= 0.985;
        b.wobble += 0.09;
        b.x += b.vx + Math.sin(b.wobble) * (b.shape === 1 ? 1.4 : 0.6);
        b.y += Math.min(b.vy, b.shape === 0 ? 3.2 : 2.4);
        b.rot += b.vr;
        ctx.save();
        ctx.translate(b.x, b.y);
        ctx.rotate(b.rot);
        ctx.fillStyle = b.color;
        if (b.shape === 0) {
          // A flake turning in the light: its width breathes as it spins.
          ctx.fillRect(-b.size / 2, -b.size / 4, b.size * Math.abs(Math.cos(b.wobble)), b.size / 2);
        } else if (b.shape === 1) {
          ctx.beginPath();
          ctx.ellipse(0, 0, b.size * 0.55, b.size, 0, 0, Math.PI * 2);
          ctx.fill();
        } else {
          heart(ctx, b.size);
        }
        ctx.restore();
      }
      frame.current = bits.current.length ? requestAnimationFrame(tick) : null;
    };
    if (frame.current === null) frame.current = requestAnimationFrame(tick);
  }, [fire, originY]);

  useEffect(
    () => () => {
      if (frame.current !== null) cancelAnimationFrame(frame.current);
    },
    [],
  );

  // Mounted a tick late by OnTop; nothing fires sooner than that.
  return (
    <OnTop>
      <canvas
        ref={canvas}
        aria-hidden="true"
        className="pointer-events-none fixed inset-0 z-[60] h-full w-full"
      />
    </OnTop>
  );
}

/* ---------- Cheers ---------- */

/** A champagne flute, drawn. `id` keeps the two glasses' gradients apart. */
function Flute({ id }: { id: string }) {
  return (
    <svg width="86" height="202" viewBox="0 0 64 150" aria-hidden="true">
      <defs>
        <linearGradient id={`${id}-wine`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#fff3cf" />
          <stop offset="1" stopColor="#d9a84e" />
        </linearGradient>
        <linearGradient id={`${id}-glass`} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor="rgba(255,255,255,0.55)" />
          <stop offset="0.5" stopColor="rgba(255,255,255,0.12)" />
          <stop offset="1" stopColor="rgba(255,255,255,0.4)" />
        </linearGradient>
      </defs>
      {/* the wine, then the glass over it */}
      <path d="M17 30 C17 62 22 82 32 86 C42 82 47 62 47 30 Z" fill={`url(#${id}-wine)`} />
      <path
        d="M14 6 C13 50 18 84 32 90 C46 84 51 50 50 6 Z"
        fill={`url(#${id}-glass)`}
        stroke="rgba(255,240,245,0.85)"
        strokeWidth="1.6"
      />
      <path d="M32 90 V136" stroke="rgba(255,240,245,0.85)" strokeWidth="2" />
      <ellipse
        cx="32"
        cy="139"
        rx="16"
        ry="4"
        fill="none"
        stroke="rgba(255,240,245,0.85)"
        strokeWidth="1.8"
      />
      {[
        [26, 70, 1.6, "0s"],
        [36, 60, 1.2, "0.6s"],
        [30, 48, 1.4, "1.1s"],
        [38, 40, 1, "0.3s"],
      ].map(([cx, cy, r, d]) => (
        <circle
          key={`${cx}-${cy}`}
          cx={cx}
          cy={cy}
          r={r}
          fill="#fffaf0"
          className="animate-twinkle"
          style={{ animationDelay: d as string }}
        />
      ))}
    </svg>
  );
}

/**
 * Two glasses meet in the middle of the screen, a spark where they touch,
 * and the toast written underneath. Closes itself, or on a tap.
 */
export function CheersMoment({
  years,
  said,
  onDone,
}: {
  years: number;
  /** What happened to the cheers sent to them, once known. */
  said: string | null;
  onDone: () => void;
}) {
  useEffect(() => {
    const timer = setTimeout(onDone, 4800);
    return () => clearTimeout(timer);
  }, [onDone]);

  return (
    <OnTop>
      <button
        type="button"
        onClick={onDone}
        aria-label="Close"
        className="fixed inset-0 z-[55] flex animate-fade flex-col items-center justify-center bg-[rgba(32,10,22,0.72)] backdrop-blur-[3px]"
      >
        <div className="relative flex items-end" aria-hidden="true">
          <div className="origin-bottom animate-clink-l">
            <Flute id="fl" />
          </div>
          <div className="-ml-3 origin-bottom animate-clink-r">
            <Flute id="fr" />
          </div>
          {/* The spark where the rims touch. */}
          <svg
            width="56"
            height="56"
            viewBox="0 0 56 56"
            className="absolute left-1/2 top-[-22px] -translate-x-1/2 animate-pop"
          >
            {[0, 45, 90, 135, 180, 225, 270, 315].map((a) => (
              <line
                key={a}
                x1="28"
                y1="28"
                x2={28 + Math.cos((a * Math.PI) / 180) * (a % 90 === 0 ? 26 : 16)}
                y2={28 + Math.sin((a * Math.PI) / 180) * (a % 90 === 0 ? 26 : 16)}
                stroke="#f6e2ae"
                strokeWidth="2"
                strokeLinecap="round"
              />
            ))}
          </svg>
        </div>
        <div className="mt-8 animate-occasion-1 font-serif text-[64px] italic leading-none">
          <span className="animate-shimmer text-foil">Cheers</span>
        </div>
        <div className="mt-3 animate-occasion-2 text-[17px] text-[#f5dbe4]">
          to {yearsWord(years).toLowerCase()}, and every one to come
        </div>
        <div role="status" className="mt-6 min-h-5 px-8 text-center text-[14px] text-[#dcb6c5]">
          {said}
        </div>
      </button>
    </OnTop>
  );
}
