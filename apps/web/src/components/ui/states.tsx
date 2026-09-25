"use client";

import { type ReactNode } from "react";
import { Icon, type IconName } from "@/components/icons";
import { Button } from "./buttons";
import { Para } from "./typography";
import { cx } from "./cx";

export function Skeleton({ lines = 3 }: { lines?: number }) {
  return (
    <div aria-hidden="true" className="flex flex-col gap-3 pt-2">
      {Array.from({ length: lines }, (_, i) => (
        <div
          key={i}
          className="h-3.5 animate-breathe rounded-lg bg-faint"
          style={{ width: `${70 - i * 14}%` }}
        />
      ))}
    </div>
  );
}

/** Empty-state illustrations shaped like what's missing (album corners, ruled
 *  lines, a goal's measure, a timeline), not a generic placeholder. */
export function Ghost({ kind = "lines" }: { kind?: "lines" | "frames" | "bars" | "dots" }) {
  if (kind === "frames") {
    // Corner mounts: the paper triangles that held a photograph into an album.
    return (
      <div aria-hidden="true" className="flex items-end gap-3.5 pb-3.5">
        <span className="relative block h-[96px] w-[118px]">
          {[
            "left-0 top-0 border-l-2 border-t-2",
            "right-0 top-0 border-r-2 border-t-2",
            "bottom-0 left-0 border-b-2 border-l-2",
            "bottom-0 right-0 border-b-2 border-r-2",
          ].map((pos) => (
            <span key={pos} className={cx("absolute h-4 w-4 border-edge", pos)} />
          ))}
        </span>
        <span className="h-[58px] w-[52px] rounded-input bg-faint opacity-45" />
      </div>
    );
  }
  if (kind === "bars") {
    // A measure with nothing on it yet: the track a goal will fill.
    return (
      <div aria-hidden="true" className="flex flex-col gap-4 pb-3.5">
        {[200, 150, 100].map((w, i) => (
          <div key={w} className="flex items-center gap-3" style={{ opacity: 1 - i * 0.35 }}>
            <span className="h-1.5 rounded-full bg-faint" style={{ width: w }} />
            <span className="h-1.5 w-1.5 rounded-full bg-faint" />
          </div>
        ))}
      </div>
    );
  }
  if (kind === "dots") {
    // The timeline History already draws, waiting for its first entry.
    return (
      <div aria-hidden="true" className="relative flex flex-col gap-[18px] pb-3.5 pl-1">
        <span className="absolute bottom-3.5 left-[4.5px] top-1.5 w-px bg-faint" />
        {[150, 120, 96].map((w, i) => (
          <div
            key={w}
            className="relative flex items-center gap-4"
            style={{ opacity: 1 - i * 0.35 }}
          >
            <span className="box-border h-[9px] w-[9px] shrink-0 rounded-full border-[1.5px] border-edge bg-bg" />
            <span className="h-2.5 rounded-full bg-faint" style={{ width: w }} />
          </div>
        ))}
      </div>
    );
  }
  // Ruled lines, for the screens that hold something written.
  return (
    <div aria-hidden="true" className="flex flex-col gap-3.5 pb-3.5">
      {[0, 1, 2, 3].map((i) => (
        <span
          key={i}
          className="h-px bg-faint"
          style={{ width: [210, 168, 190, 104][i], opacity: 1 - i * 0.2 }}
        />
      ))}
    </div>
  );
}

export function EmptyState({
  ghost,
  title,
  text,
  cta,
}: {
  ghost: "lines" | "frames" | "bars" | "dots";
  title: string;
  text: string;
  cta: ReactNode;
}) {
  return (
    <div className="flex grow flex-col justify-center gap-3.5 pb-12">
      <Ghost kind={ghost} />
      <h2 className="m-0 text-[26px] font-semibold leading-tight tracking-[-0.02em]">{title}</h2>
      <Para size="lg">{text}</Para>
      <div className="mt-3">{cta}</div>
    </div>
  );
}

/** Inline failure notice, shown in place of just the missing content.
 *  `tone="error"` is a real failure worth retrying; `quiet` needs no alarm. */
export function LoadProblem({
  icon,
  title,
  text,
  tone,
  onRetry,
}: {
  icon: IconName;
  title: string;
  text?: string;
  tone: "error" | "quiet";
  onRetry?: () => void;
}) {
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className="my-2 flex items-center gap-3 rounded-card bg-paper py-2.5 pl-4 pr-1.5"
    >
      <Icon name={icon} size={20} className={tone === "error" ? "text-red" : "text-stone"} />
      <div className="flex grow flex-col py-0.5">
        <span className="text-[15px] font-semibold text-ink">{title}</span>
        {text ? <span className="text-support text-stone">{text}</span> : null}
      </div>
      {onRetry ? (
        <button
          type="button"
          onClick={onRetry}
          className="press h-11 shrink-0 px-2.5 text-[15px] font-semibold text-plum"
        >
          Try again
        </button>
      ) : null}
    </div>
  );
}

/** Whole-screen failure. Only for when nothing at all can be shown: the app shell, a crashed route. */
export function ErrorState({
  title,
  text,
  onRetry,
  secondary,
}: {
  title: string;
  text: string;
  onRetry: () => void;
  secondary?: ReactNode;
}) {
  return (
    <div role="alert" className="flex grow flex-col justify-center gap-3.5 pb-10">
      <Icon name="alert" size={28} strokeWidth={1.4} className="text-red" />
      <h2 className="m-0 text-[24px] font-semibold leading-tight tracking-[-0.02em]">{title}</h2>
      <Para>{text}</Para>
      <div className="mt-2.5 flex flex-col gap-1">
        <Button icon="sync" onClick={onRetry}>
          Try again
        </Button>
        {secondary}
      </div>
    </div>
  );
}

export function Alert({
  message,
  variant = "error",
}: {
  message: string;
  variant?: "error" | "success";
}) {
  const tone = variant === "error" ? "bg-red-tint text-red" : "bg-green-tint text-green";
  return (
    <div
      role={variant === "error" ? "alert" : "status"}
      className={`flex items-center gap-2.5 rounded-input px-3.5 py-2.5 text-support font-semibold ${tone}`}
    >
      <Icon name={variant === "error" ? "alert" : "check"} size={18} />
      {message}
    </div>
  );
}
