"use client";

import type { ReactNode } from "react";
import { cx } from "@/components/ui/kit";

/**
 * One screen = one column that fills the dynamic viewport.
 * Top padding respects the iOS status bar in standalone mode. Content scrolls inside `main`
 * so the tab bar and bottom actions stay put.
 *
 * `size="app"` is the signed-in column: it grows with the viewport, but only to
 * a comfortable measure — text is not more readable at 1200px, so the width a
 * tablet or desktop adds goes to the side nav and to side-by-side sections
 * (see the Together hub), never to longer lines. A form or a legal page keeps
 * the phone width at every size, centred.
 */
export function Screen({
  children,
  className,
  tone = "bg",
  size = "narrow",
}: {
  children: ReactNode;
  className?: string;
  tone?: "bg" | "paper";
  size?: "narrow" | "app";
}) {
  return (
    <div
      className={cx(
        "mx-auto flex min-h-[100dvh] w-full flex-col",
        size === "app" ? "max-w-[520px] md:max-w-[680px] lg:max-w-[760px]" : "max-w-[520px]",
        tone === "paper" ? "bg-paper" : "bg-bg",
        // On the same element the grain paints over the fill, so the column
        // keeps one background and gains a surface. It scrolls with the page,
        // because paper does.
        "grain",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function SafeTop({ className }: { className?: string }) {
  return (
    <div className={cx("shrink-0", className)} style={{ height: "calc(var(--safe-top) + 42px)" }} />
  );
}

export function Main({
  children,
  className,
  pad = true,
}: {
  children: ReactNode;
  className?: string;
  pad?: boolean;
}) {
  return (
    <main className={cx("flex grow animate-page flex-col", pad && "px-6 md:px-8", className)}>
      {children}
    </main>
  );
}
