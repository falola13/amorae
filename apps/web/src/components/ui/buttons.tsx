"use client";

import Link from "next/link";
import { type ButtonHTMLAttributes, type ReactNode } from "react";
import { Icon, type IconName } from "@/components/icons";
import { cx } from "./cx";

/* ---------- Buttons ---------- */

export type Variant = "primary" | "secondary" | "text" | "danger" | "quiet";

export const VARIANT: Record<Variant, string> = {
  primary: "h-[54px] w-full rounded-btn bg-plum text-surface",
  secondary: "h-[54px] w-full rounded-btn border border-edge bg-transparent text-ink",
  text: "h-11 w-full rounded-btn bg-transparent text-plum",
  danger: "h-11 w-full rounded-btn bg-transparent text-red",
  // Ink, not plum: the step-through controls (Previous / Next / Keep it) sit
  // beside the one plum action and must not compete with it. Sized to its
  // words, so a pair can share a line.
  quiet: "h-12 w-auto rounded-btn bg-transparent px-2 text-[15px] text-ink",
};

// Width lives in VARIANT, so `quiet` can be content-sized without fighting a w-full here.
export const BASE =
  "press inline-flex items-center justify-center gap-2 text-[16px] font-semibold disabled:opacity-100 disabled:bg-faint disabled:text-stone";

export function buttonClass(variant: Variant = "primary", className?: string) {
  return cx(BASE, VARIANT[variant], className);
}

export function Button({
  variant = "primary",
  icon,
  loading,
  children,
  className,
  ...rest
}: {
  variant?: Variant;
  icon?: IconName;
  loading?: boolean;
} & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      type="button"
      aria-busy={loading || undefined}
      className={buttonClass(variant, className)}
      disabled={loading || rest.disabled}
      {...rest}
    >
      {loading ? (
        <Spinner light={variant === "primary"} />
      ) : icon ? (
        <Icon name={icon} size={20} />
      ) : null}
      {children}
    </button>
  );
}

export function LinkButton({
  href,
  variant = "primary",
  icon,
  children,
  className,
  replace,
}: {
  href: string;
  variant?: Variant;
  icon?: IconName;
  children: ReactNode;
  className?: string;
  replace?: boolean;
}) {
  return (
    <Link
      href={href}
      replace={replace}
      className={buttonClass(variant, cx("no-underline", className))}
    >
      {icon ? <Icon name={icon} size={20} /> : null}
      {children}
    </Link>
  );
}

export function Spinner({ light }: { light?: boolean }) {
  return (
    <span
      className={cx(
        "inline-block h-[18px] w-[18px] animate-spin rounded-full border-2",
        light ? "border-plum-tint border-t-surface" : "border-faint border-t-plum",
      )}
    />
  );
}
