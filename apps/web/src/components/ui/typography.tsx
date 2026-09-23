"use client";

import { type ReactNode } from "react";
import { cx } from "./cx";

/* ---------- Type ---------- */

export function Micro({
  children,
  className,
  tone = "stone",
}: {
  children: ReactNode;
  className?: string;
  tone?: "stone" | "plum";
}) {
  return (
    <div
      className={cx(
        "text-micro uppercase",
        tone === "plum" ? "text-plum" : "text-stone",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function Title({
  children,
  className,
  as: As = "h1",
  size = "title",
}: {
  children: ReactNode;
  className?: string;
  as?: "h1" | "h2";
  size?: "title" | "display" | "lg";
}) {
  const s =
    size === "display"
      ? "text-display"
      : size === "lg"
        ? "text-[32px] leading-[1.15] font-semibold tracking-[-0.025em]"
        : "text-title";
  return <As className={cx("m-0", s, className)}>{children}</As>;
}

export function Para({
  children,
  className,
  size = "body",
}: {
  children: ReactNode;
  className?: string;
  size?: "body" | "lg" | "support";
}) {
  return (
    <p
      className={cx(
        "m-0 text-stone",
        size === "lg" ? "text-bodylg" : size === "support" ? "text-support" : "text-body",
        className,
      )}
    >
      {children}
    </p>
  );
}
