import { cx } from "@/components/ui/kit";

const DAYS = [
  { value: 0, short: "S", full: "Sunday" },
  { value: 1, short: "M", full: "Monday" },
  { value: 2, short: "T", full: "Tuesday" },
  { value: 3, short: "W", full: "Wednesday" },
  { value: 4, short: "T", full: "Thursday" },
  { value: 5, short: "F", full: "Friday" },
  { value: 6, short: "S", full: "Saturday" },
] as const;

const chipClass = (active: boolean, wide = false) =>
  cx(
    "press flex h-9 shrink-0 items-center justify-center rounded-full text-[14px] font-semibold",
    wide ? "px-3.5" : "w-9",
    active ? "bg-plum-tint text-plum" : "text-stone",
  );

/**
 * "Every day" plus the seven weekdays, mutually exclusive with each other: picking a day drops
 * "Every day", and dropping the last picked day naturally lands back on `[]` — which already
 * means every day, so there is nothing special to do for it.
 */
export function WeekdayChips({
  value,
  onChange,
}: {
  value: number[];
  onChange: (next: number[]) => void;
}) {
  const everyDay = value.length === 0;
  const toggle = (d: number) => {
    if (everyDay) {
      onChange([d]);
    } else if (value.includes(d)) {
      onChange(value.filter((x) => x !== d));
    } else {
      onChange([...value, d].sort((a, b) => a - b));
    }
  };

  return (
    <div role="group" aria-label="Which days?" className="-mx-1 flex flex-wrap gap-1">
      <button
        type="button"
        aria-pressed={everyDay}
        onClick={() => onChange([])}
        className={chipClass(everyDay, true)}
      >
        Every day
      </button>
      {DAYS.map((d) => (
        <button
          key={d.value}
          type="button"
          aria-pressed={!everyDay && value.includes(d.value)}
          aria-label={d.full}
          onClick={() => toggle(d.value)}
          className={chipClass(!everyDay && value.includes(d.value))}
        >
          {d.short}
        </button>
      ))}
    </div>
  );
}
