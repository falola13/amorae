import { cx } from "./cx";

/** Not the logo — brand guidelines keep the mark ≥16px and never decorative. */
function EndMark({ className }: { className?: string }) {
  return (
    <svg
      width="26"
      height="11"
      viewBox="0 0 26 11"
      aria-hidden="true"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.3}
      strokeLinecap="round"
      className={cx("shrink-0", className)}
    >
      <path d="M1 9.2C5.9 9.2 9.2 5.8 11.2 1.8" />
      <path d="M25 9.2C20.1 9.2 16.8 5.8 14.8 1.8" />
    </svg>
  );
}

/** `caption` should state a fact, not decorate. */
export function Ornament({ caption, className }: { caption?: string; className?: string }) {
  return (
    <div className={cx("flex flex-col items-center gap-3 pb-2 pt-9", className)}>
      <div className="flex w-full max-w-[260px] items-center gap-3.5 text-faint">
        <span className="h-px grow bg-linear-to-r from-transparent to-line" />
        <EndMark />
        <span className="h-px grow bg-linear-to-l from-transparent to-line" />
      </div>
      {caption ? (
        <p className="m-0 text-center text-[13px] leading-[1.5] text-stone">{caption}</p>
      ) : null}
    </div>
  );
}
