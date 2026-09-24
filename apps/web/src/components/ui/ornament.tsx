import { cx } from "./cx";

/**
 * A printer's end-mark, for the foot of an archive.
 *
 * Deliberately not the logo. The brand keeps the mark at 16px and up and asks
 * that it not be dropped into rings or repeated as furniture — and a mark used
 * as decoration stops being a mark. This is the idea underneath it instead: two
 * strokes curving toward a centre they share, and not quite reaching it. Same
 * thought, ornament weight, no trademark spent.
 *
 * The rules either side fade out rather than stopping, which is what a rule
 * does on a printed page and what a hairline `border-line` cannot do.
 */
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

/**
 * The end of something worth having scrolled through.
 *
 * `caption` is for a true sentence, not a flourish — how many moments are kept,
 * when the first week was. A line that states a fact earns the space at the
 * bottom of a short screen; a line that only decorates it does not, and reads
 * as padding the moment you notice it twice.
 */
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
