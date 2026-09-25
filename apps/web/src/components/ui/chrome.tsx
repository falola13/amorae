"use client";

import Link from "next/link";
import { type ReactNode } from "react";
import { Icon, type IconName } from "@/components/icons";
import { useDialog } from "@/lib/hooks/use-dialog";
import { Spinner } from "./buttons";
import { cx } from "./cx";

/* ---------- Chrome ---------- */

export function TopBar({
  back,
  backHref,
  right,
  center,
  onBack,
}: {
  back?: string;
  backHref?: string;
  right?: ReactNode;
  center?: ReactNode;
  onBack?: () => void;
}) {
  const cls =
    "press flex h-11 items-center gap-0.5 pl-1 pr-2 text-[16px] font-medium text-ink no-underline";
  return (
    <div className="flex h-11 shrink-0 items-center justify-between px-3">
      {backHref ? (
        <Link href={backHref} className={cls}>
          <Icon name="left" size={22} />
          {back}
        </Link>
      ) : onBack ? (
        <button type="button" onClick={onBack} className={cls}>
          <Icon name="left" size={22} />
          {back}
        </button>
      ) : (
        <span />
      )}
      {center ? <div className="text-[13px] font-semibold text-stone">{center}</div> : null}
      {right ?? <span />}
    </div>
  );
}

/** Compose bar: Cancel and Save stay in the top bar, reachable with the keyboard open. */
export function ComposeBar({
  cancelHref,
  label,
  done = "Save",
  onDone,
  onCancel,
  busy,
}: {
  cancelHref?: string;
  label: string;
  done?: string;
  onDone?: () => void;
  onCancel?: () => void;
  busy?: boolean;
}) {
  const c = "press flex h-11 items-center px-3 text-[16px] font-medium text-ink no-underline";
  return (
    <div className="flex h-11 shrink-0 items-center justify-between px-3">
      {cancelHref ? (
        <Link href={cancelHref} className={c}>
          Cancel
        </Link>
      ) : (
        <button type="button" onClick={onCancel} className={c}>
          Cancel
        </button>
      )}
      <div className="text-[13px] font-semibold text-stone">{label}</div>
      <button
        type="button"
        onClick={onDone}
        disabled={busy}
        className="press flex h-11 items-center px-3 text-[16px] font-bold text-plum disabled:text-stone"
      >
        {busy ? <Spinner /> : done}
      </button>
    </div>
  );
}

/** Sticky action area above the tab bar or the home indicator. */
export function BottomActions({
  children,
  className,
  safe,
}: {
  children: ReactNode;
  className?: string;
  safe?: boolean;
}) {
  return (
    <div
      className={cx("flex shrink-0 flex-col gap-1 px-6 pt-3", safe ? "pb-safe" : "pb-3", className)}
    >
      {children}
    </div>
  );
}

export function Banner({
  tone,
  icon,
  children,
}: {
  tone: "amber" | "green";
  icon: IconName;
  children: ReactNode;
}) {
  const t = tone === "amber" ? "bg-amber-tint text-amber" : "bg-green-tint text-green";
  return (
    <div
      role="status"
      className={cx(
        "mx-4 flex animate-rise items-center gap-2.5 rounded-input px-3.5 py-2.5 text-support font-semibold",
        t,
      )}
    >
      <Icon name={icon} size={18} strokeWidth={1.7} />
      {children}
    </div>
  );
}

export function Sheet({
  open,
  title,
  onClose,
  children,
  labelledBy,
}: {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
  labelledBy: string;
}) {
  const dialogRef = useDialog(open, onClose);
  if (!open) return null;
  // Bottom sheet on a phone; centred dialog from `md`.
  return (
    <div className="fixed inset-0 z-40 md:flex md:items-center md:justify-center md:p-6">
      <button
        type="button"
        aria-label="Close"
        onClick={onClose}
        className="absolute inset-0 animate-fade bg-ink/40"
      />
      <div
        ref={dialogRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        className="absolute inset-x-0 bottom-0 flex animate-sheet flex-col gap-4 rounded-t-sheet bg-bg px-6 pt-2.5 pb-safe md:static md:w-full md:max-w-[440px] md:rounded-card md:border md:border-line md:p-7"
      >
        <div aria-hidden="true" className="h-1 w-9 self-center rounded-full bg-faint md:hidden" />
        <h2
          id={labelledBy}
          className="m-0 mt-2 text-[24px] font-semibold leading-tight tracking-[-0.02em] md:mt-0"
        >
          {title}
        </h2>
        {children}
      </div>
    </div>
  );
}

export function Toast({
  message,
  action,
  onAction,
}: {
  message: string;
  action?: string;
  onAction?: () => void;
}) {
  // Above the tab bar on a phone; centred near the bottom edge from `md`.
  return (
    <div
      role="status"
      className="fixed inset-x-6 bottom-[calc(92px+var(--safe-bottom))] z-50 flex h-[52px] animate-rise items-center justify-between gap-2.5 rounded-btn bg-ink pl-4 pr-2 text-[15px] font-medium text-surface md:inset-x-auto md:bottom-8 md:left-1/2 md:w-[420px] md:-translate-x-1/2"
    >
      {message}
      {action ? (
        <button
          type="button"
          onClick={onAction}
          className="press h-11 px-2.5 text-[15px] font-bold text-surface"
        >
          {action}
        </button>
      ) : null}
    </div>
  );
}
