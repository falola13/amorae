import Link from "next/link";
import { dayNum, shortMonth } from "@/lib/dates";
import { Icon } from "@/components/icons";
import { cx } from "@/components/ui/kit";

/** Event and milestone rows: big day number, short month, then the words. */
export function DateRow({
  date,
  title,
  sub,
  href,
  past,
  action,
  right,
  onAction,
  last,
}: {
  date: string;
  title: string;
  sub?: string;
  href?: string;
  past?: boolean;
  action?: string;
  right?: string;
  /**
   * What else can be done to this row, behind a button at its end. Only for
   * rows that are not themselves a link: a button inside a link is a target
   * inside a target, and on a phone that is a coin toss.
   */
  onAction?: () => void;
  last?: boolean;
}) {
  const inner = (
    <>
      <span className="flex w-10 shrink-0 flex-col items-center">
        <span
          className={cx(
            "tabular text-[22px] font-semibold leading-[1.1] tracking-[-0.02em]",
            past ? "text-stone" : "text-ink",
          )}
        >
          {dayNum(date)}
        </span>
        <span className="text-[11px] font-bold uppercase tracking-[0.08em] text-stone">
          {shortMonth(date)}
        </span>
      </span>
      <span className="flex grow flex-col gap-px">
        <span className="text-bodylg font-semibold tracking-[-0.01em] text-ink">{title}</span>
        {sub ? <span className="text-support text-stone">{sub}</span> : null}
        {action ? <span className="text-support font-semibold text-plum">{action}</span> : null}
      </span>
      {right ? (
        <span className="shrink-0 text-support font-semibold text-plum">{right}</span>
      ) : href ? (
        <Icon name="right" size={18} className="text-stone" />
      ) : null}
      {onAction && !href ? (
        <button
          type="button"
          onClick={onAction}
          aria-label={`What to do with ${title}`}
          className="press -mr-2.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-btn text-stone"
        >
          <Icon name="more" size={20} />
        </button>
      ) : null}
    </>
  );
  const cls = cx(
    "press flex min-h-[76px] items-center gap-4 no-underline",
    !last && "border-b border-line",
  );
  return href ? (
    <Link href={href} className={cls}>
      {inner}
    </Link>
  ) : (
    <div className={cls}>{inner}</div>
  );
}
