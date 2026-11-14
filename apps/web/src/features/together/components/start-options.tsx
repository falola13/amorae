"use client";

import { Field, Segmented } from "@/components/ui/kit";
import type { ChallengeKind } from "@/lib/api/types";
import { iso } from "@/lib/dates";
import { today } from "@/lib/today";
import { startBounds } from "../challenges";

export type StartWhen = "today" | "pick";

const KINDS: { value: ChallengeKind; label: string }[] = [
  { value: "together", label: "Both of us" },
  { value: "mine", label: "Just me" },
];

/** "Who’s taking part": both of you, or just you. */
export function KindChips({
  value,
  onChange,
  disabled,
}: {
  value: ChallengeKind;
  onChange: (k: ChallengeKind) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <div className="text-support font-semibold text-stone">Who’s taking part</div>
      <Segmented
        label="Who’s taking part"
        options={KINDS}
        value={value}
        onChange={onChange}
        disabled={disabled}
      />
    </div>
  );
}

/** "Starts": today, or a day up to sixty days out. */
export function StartsChips({
  when,
  onWhen,
  date,
  onDate,
  error,
}: {
  when: StartWhen;
  onWhen: (w: StartWhen) => void;
  date: string;
  onDate: (d: string) => void;
  error?: string;
}) {
  const { min, max } = startBounds(today());
  const options: { value: StartWhen; label: string }[] = [
    { value: "today", label: "Today" },
    { value: "pick", label: "Pick a day" },
  ];
  return (
    <div className="flex flex-col gap-2">
      <div className="text-support font-semibold text-stone">Starts</div>
      <Segmented label="Starts" options={options} value={when} onChange={onWhen} />
      {when === "pick" ? (
        <Field
          label="Start date"
          type="date"
          min={min}
          max={max}
          value={date}
          error={error}
          onChange={(e) => onDate(e.target.value)}
          className="w-52"
        />
      ) : null}
    </div>
  );
}

/** Tomorrow, the usual first pick when choosing a day. */
export const tomorrow = (): string => {
  const d = today();
  d.setDate(d.getDate() + 1);
  return iso(d);
};
