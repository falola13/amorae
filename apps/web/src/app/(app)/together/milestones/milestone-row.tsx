import { dayNum, shortMonth } from "@/lib/dates";
import { Icon, type IconName } from "@/components/icons";
import { cx } from "@/components/ui/kit";
import type { Milestone } from "@/lib/api/types";

/** A small uppercase pill for "Today" (filled, plum) or "Soon" (quiet, stone). */
function Chip({ tone, children }: { tone: "today" | "soon"; children: string }) {
  return (
    <span
      className={cx(
        "shrink-0 rounded-full px-2 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em]",
        tone === "today" ? "bg-plum text-surface" : "bg-faint text-stone",
      )}
    >
      {children}
    </span>
  );
}

const sourceIcon = (source: Milestone["source"]): IconName | null =>
  source === "birthday" ? "cake" : source === "anniversary" ? "heart" : null;

/** Like DateRow, with a source icon (cake/heart) and a "Today"/"Soon" chip —
 *  drawn locally instead of extending the shared DateRow, since this design
 *  is specific to important dates. */
export function MilestoneRow({
  date,
  title,
  sub,
  source,
  isToday,
  isSoon,
  action,
  right,
  onAction,
  last,
}: {
  date: string;
  title: string;
  sub?: string;
  source: Milestone["source"];
  isToday?: boolean;
  isSoon?: boolean;
  action?: string;
  right?: string;
  onAction?: () => void;
  last?: boolean;
}) {
  const icon = sourceIcon(source);
  return (
    <div
      className={cx(
        "flex min-h-[76px] items-center gap-4",
        isToday ? "-mx-3 rounded-card bg-celebrate px-3" : !last && "border-b border-line",
      )}
    >
      <span className="flex w-10 shrink-0 flex-col items-center gap-0.5">
        {icon ? <Icon name={icon} size={16} className="text-plum" /> : null}
        <span className="tabular text-[22px] font-semibold leading-[1.1] tracking-[-0.02em] text-ink">
          {dayNum(date)}
        </span>
        <span className="text-[11px] font-bold uppercase tracking-[0.08em] text-stone">
          {shortMonth(date)}
        </span>
      </span>
      <span className="flex grow flex-col gap-px">
        <span className="flex items-center gap-2">
          <span className="text-bodylg font-semibold tracking-[-0.01em] text-ink">{title}</span>
          {isToday ? (
            <Chip tone="today">Today</Chip>
          ) : isSoon ? (
            <Chip tone="soon">Soon</Chip>
          ) : null}
        </span>
        {sub ? <span className="text-support text-stone">{sub}</span> : null}
        {action ? <span className="text-support font-semibold text-plum">{action}</span> : null}
      </span>
      {right ? (
        <span className="shrink-0 text-support font-semibold text-plum">{right}</span>
      ) : null}
      {onAction ? (
        <button
          type="button"
          onClick={onAction}
          aria-label={`What to do with ${title}`}
          className="press -mr-2.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-btn text-stone"
        >
          <Icon name="more" size={20} />
        </button>
      ) : null}
    </div>
  );
}
