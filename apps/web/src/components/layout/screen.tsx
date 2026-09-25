"use client";

import type { ReactNode } from "react";
import { cx } from "@/components/ui/kit";

/** One screen = one column filling the dynamic viewport. `size="app"` caps line
 *  width even on tablet/desktop; the extra space goes to the side nav instead. */
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
