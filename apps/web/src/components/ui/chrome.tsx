"use client";

import Link from "next/link";
import { type PointerEvent as ReactPointerEvent, type ReactNode, useRef, useState } from "react";
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

type SheetProps = {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
  labelledBy: string;
  /** Something typed and not saved. A stray tap outside or Escape then asks
   *  before throwing it away; the sheet's own Cancel still closes at once. */
  dirty?: boolean;
};

export function Sheet(props: SheetProps) {
  // Mounted only while open, so the "discard?" question never outlives one opening.
  return props.open ? <OpenSheet {...props} /> : null;
}

function OpenSheet({ title, onClose, children, labelledBy, dirty = false }: SheetProps) {
  const [asking, setAsking] = useState(false);
  const dismiss = () => (dirty ? setAsking(true) : onClose());
  const dialogRef = useDialog(true, dismiss);

  // The handle promised a swipe down and there wasn't one. The handle and the
  // title are the grip: far enough, or quick enough, and it goes — through
  // dismiss, so unsaved words still get asked about rather than lost.
  const [drag, setDrag] = useState(0);
  const [dragging, setDragging] = useState(false);
  const from = useRef<{ y: number; t: number } | null>(null);
  const grip = {
    onPointerDown: (e: ReactPointerEvent<HTMLDivElement>) => {
      if (e.pointerType === "mouse" && e.button !== 0) return;
      from.current = { y: e.clientY, t: e.timeStamp };
      e.currentTarget.setPointerCapture(e.pointerId);
      setDragging(true);
    },
    onPointerMove: (e: ReactPointerEvent<HTMLDivElement>) => {
      if (from.current) setDrag(Math.max(0, e.clientY - from.current.y));
    },
    onPointerUp: (e: ReactPointerEvent<HTMLDivElement>) => {
      const f = from.current;
      from.current = null;
      setDragging(false);
      setDrag(0);
      if (!f) return;
      const dy = Math.max(0, e.clientY - f.y);
      const speed = dy / Math.max(1, e.timeStamp - f.t);
      if (dy > 120 || (dy > 40 && speed > 0.6)) dismiss();
    },
    onPointerCancel: () => {
      from.current = null;
      setDragging(false);
      setDrag(0);
    },
  };

  // Bottom sheet on a phone; centred dialog from `md`.
  return (
    <div className="fixed inset-0 z-40 md:flex md:items-center md:justify-center md:p-6">
      <button
        type="button"
        aria-label="Close"
        onClick={dismiss}
        className="absolute inset-0 animate-fade bg-ink/40"
      />
      <div
        ref={dialogRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        // --kb is the keyboard iOS slides over the page (useKeyboardInset):
        // sit on top of it, and scroll inside when there isn't room for all.
        // `translate`, not `transform`: the entrance animation holds its last
        // transform frame, and an animation beats an inline style.
        style={{
          translate: drag ? `0 ${drag}px` : undefined,
          transition: dragging ? "none" : "translate 0.2s ease-out",
        }}
        className="absolute inset-x-0 bottom-[var(--kb,0px)] flex max-h-[calc(100dvh-var(--kb,0px)-24px)] animate-sheet flex-col gap-4 overflow-y-auto overscroll-contain rounded-t-sheet bg-bg px-6 pb-safe md:static md:max-h-[calc(100dvh-48px)] md:w-full md:max-w-[440px] md:rounded-card md:border md:border-line md:p-7"
      >
        <div {...grip} className="-mx-6 flex touch-none flex-col px-6 pt-2.5 md:contents">
          <div aria-hidden="true" className="h-1 w-9 self-center rounded-full bg-faint md:hidden" />
          <h2
            id={labelledBy}
            className="m-0 mt-6 text-[24px] font-semibold leading-tight tracking-[-0.02em] md:mt-0"
          >
            {title}
          </h2>
        </div>
        {asking ? (
          <div
            role="alert"
            className="flex items-center justify-between gap-2 rounded-input bg-faint px-3.5 py-2 text-support font-semibold"
          >
            Discard what you typed?
            <span className="flex shrink-0 gap-1">
              <button
                type="button"
                onClick={() => setAsking(false)}
                className="press h-11 whitespace-nowrap px-2.5 text-ink"
              >
                Keep it
              </button>
              <button
                type="button"
                onClick={onClose}
                className="press h-11 whitespace-nowrap px-2.5 text-red"
              >
                Discard
              </button>
            </span>
          </div>
        ) : null}
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
