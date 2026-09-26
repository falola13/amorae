import { cx } from "@/components/ui/kit";
import type { PrayerDay } from "@/lib/api/types";
import { dayName } from "@/lib/dates";

type Mark = "full" | "faint" | "dash";

const markFor = (day: PrayerDay, who: "mine" | "partner"): Mark => {
  if (day.points.length === 0) return "dash";
  const marked = day[who];
  return day.points.every((id) => marked.includes(id)) ? "full" : "faint";
};

const dayLabel = (day: PrayerDay, partnerName: string): string => {
  const name = dayName(day.date);
  if (day.points.length === 0) return `${name}: nothing set`;
  const mine = markFor(day, "mine") === "full";
  const partner = markFor(day, "partner") === "full";
  return `${name}: ${mine ? "you prayed" : "you didn’t"}, ${
    partner ? `${partnerName} prayed` : `${partnerName} didn’t`
  }`;
};

function Dot({ mark }: { mark: Mark }) {
  if (mark === "dash") {
    return (
      <span aria-hidden="true" className="text-[10px] leading-none text-edge">
        &ndash;
      </span>
    );
  }
  return (
    <span
      aria-hidden="true"
      className={cx(
        "block h-[7px] w-[7px] rounded-full",
        mark === "full" ? "bg-plum" : "border border-edge",
      )}
    />
  );
}

/**
 * A quiet week-at-a-glance: for each day, whether the two of you prayed everything set for it —
 * no streaks or scores, just today plus the six around it. `today` is only present for the
 * current week (from PrayerWeek.today), so a week from history renders with nothing dimmed.
 */
export function DaysRow({
  days,
  today,
  partnerName,
}: {
  days: PrayerDay[];
  today?: string;
  partnerName: string;
}) {
  return (
    <div className="flex justify-between gap-1">
      {days.map((day) => {
        const isToday = today === day.date;
        const future = today !== undefined && day.date > today;
        return (
          <div
            key={day.date}
            role="img"
            aria-label={dayLabel(day, partnerName)}
            className={cx(
              "flex flex-col items-center gap-1.5 rounded-full px-1.5 py-1.5",
              isToday && "bg-plum-tint",
              future && "opacity-40",
            )}
          >
            <span
              aria-hidden="true"
              className={cx(
                "text-[11px] font-semibold uppercase",
                isToday ? "text-plum" : "text-stone",
              )}
            >
              {dayName(day.date)[0]}
            </span>
            <span aria-hidden="true" className="flex flex-col items-center gap-1">
              <Dot mark={markFor(day, "mine")} />
              <Dot mark={markFor(day, "partner")} />
            </span>
          </div>
        );
      })}
    </div>
  );
}
