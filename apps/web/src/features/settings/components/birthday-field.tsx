"use client";

import { useState } from "react";
import { Field, Select } from "@/components/ui/kit";
import { Icon } from "@/components/icons";
import { maxDayInMonth } from "@/lib/api/schemas";
import type { Birthday } from "@/lib/api/types";

const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];
const MONTH_OPTIONS = [
  { value: "", label: "Month" },
  ...MONTHS.map((label, i) => ({ value: String(i + 1), label })),
];

/**
 * Month + day + optional year, saved together as one field. Picking a month
 * narrows the day options to what that month (and year, if known) actually
 * has — the 29th of February stays open until a year rules it out.
 */
export function BirthdayField({
  value,
  onChange,
  hint,
  error,
}: {
  value: Birthday | null;
  onChange: (b: Birthday | null) => void;
  hint?: string;
  error?: string;
}) {
  // The form only holds a whole birthday or none, so a month picked before
  // its day has to live here — reading it back from `value` wiped it, and the
  // month select snapped back to empty the moment anything was chosen.
  const [draft, setDraft] = useState<{
    month: number | null;
    day: number | null;
    year: number | null;
  }>({ month: value?.month ?? null, day: value?.day ?? null, year: value?.year ?? null });
  // `seen` is the last value this field sent or was given. A value that
  // differs from it came from outside (loaded, reset after save) and replaces
  // the draft — adjusted during render rather than in an effect.
  const [seen, setSeen] = useState(value);
  if (value !== seen) {
    setSeen(value);
    setDraft({ month: value?.month ?? null, day: value?.day ?? null, year: value?.year ?? null });
  }
  const { month, day, year } = draft;

  const dayOptions = [
    { value: "", label: "Day" },
    ...Array.from({ length: month ? maxDayInMonth(month, year) : 31 }, (_, i) => ({
      value: String(i + 1),
      label: String(i + 1),
    })),
  ];

  // Only a month and a day together make a birthday; either missing clears it.
  const set = (patch: { month?: number | null; day?: number | null; year?: number | null }) => {
    const m = patch.month !== undefined ? patch.month : month;
    const y = patch.year !== undefined ? patch.year : year;
    let d = patch.day !== undefined ? patch.day : day;
    if (m && d) d = Math.min(d, maxDayInMonth(m, y));
    const next = m && d ? { month: m, day: d, year: y } : null;
    setDraft({ month: m, day: d, year: y });
    setSeen(next); // ours, so the check above leaves the draft alone
    onChange(next);
  };

  return (
    <div className="flex flex-col gap-2">
      <span className="text-[13px] font-semibold text-stone">Birthday</span>
      <div className="flex gap-3">
        <div className="flex-[1.3]">
          <Select
            label="Month"
            value={month ? String(month) : ""}
            onChange={(e) => set({ month: e.target.value ? Number(e.target.value) : null })}
            options={MONTH_OPTIONS}
          />
        </div>
        <div className="flex-1">
          <Select
            label="Day"
            value={day ? String(day) : ""}
            onChange={(e) => set({ day: e.target.value ? Number(e.target.value) : null })}
            options={dayOptions}
          />
        </div>
      </div>
      <Field
        label="Year (optional)"
        type="number"
        inputMode="numeric"
        placeholder="1996"
        value={year ?? ""}
        onChange={(e) => set({ year: e.target.value ? Number(e.target.value) : null })}
      />
      {error ? (
        <div role="alert" className="flex items-center gap-1.5 text-[13px] text-red">
          <Icon name="alert" size={16} />
          {error}
        </div>
      ) : hint ? (
        <div className="text-[13px] text-stone">{hint}</div>
      ) : null}
      {value ? (
        <button
          type="button"
          onClick={() => {
            setDraft({ month: null, day: null, year: null });
            setSeen(null);
            onChange(null);
          }}
          className="press self-start text-[13px] font-semibold text-plum"
        >
          Remove
        </button>
      ) : null}
    </div>
  );
}
