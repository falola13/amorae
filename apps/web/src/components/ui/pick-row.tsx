"use client";

import { useId, useState, type KeyboardEvent, type MouseEvent } from "react";
import { Icon, type IconName } from "@/components/icons";
import { cx } from "@/components/ui/kit";
import { longDate, time12 } from "@/lib/dates";

/**
 * Real native date/time/select inputs (correct picker + a11y per platform),
 * layered invisibly over the row so the app can show its own formatted text
 * ("24 September") instead of the browser's. openPicker calls showPicker()
 * on every click since focus alone won't open a desktop date dropdown; iOS
 * and Android open their native picker on focus regardless.
 */
function openPicker(event: MouseEvent<HTMLInputElement>) {
  const input = event.currentTarget;
  if (input.disabled || (input.type !== "date" && input.type !== "time")) return;
  try {
    input.showPicker();
  } catch {
    /* Older browser, or not a real gesture — the click still focuses the field. */
  }
}

/** What the app shows for a value, in its own words rather than the browser's. */
function shown(
  value: string,
  type: PickType | undefined,
  options: Options | undefined,
  placeholder: string | undefined,
) {
  if (options) return options.find((o) => o.value === value)?.label ?? value ?? placeholder ?? "";
  if (!value) return placeholder ?? "";
  if (type === "date") return longDate(value);
  if (type === "time") return time12(value);
  return value;
}

type PickType = "date" | "time" | "text";
type Options = readonly { value: string; label: string }[];

export function PickRow({
  icon,
  label,
  value,
  display,
  type,
  options,
  onChange,
  placeholder,
  disabled,
  last,
}: {
  icon: IconName;
  label: string;
  value: string;
  /** Override for values that shouldn't be read raw, e.g. 500000 as ₦500,000. */
  display?: string;
  type?: PickType;
  options?: Options;
  onChange?: (v: string) => void;
  placeholder?: string;
  disabled?: boolean;
  last?: boolean;
}) {
  const id = useId();
  // null follows `value`; a string once the user is typing their own keystrokes.
  const [draft, setDraft] = useState<string | null>(null);
  const text = draft ?? value;
  const commit = () => {
    if (draft !== null && draft !== value) onChange?.(draft);
    setDraft(null);
  };
  const native = "absolute inset-0 h-full w-full cursor-pointer opacity-0 disabled:cursor-default";

  return (
    <label
      htmlFor={id}
      className={cx(
        "relative flex h-[54px] w-full items-center gap-3.5 text-[16px] font-medium text-ink",
        "rounded-[10px] focus-within:outline focus-within:outline-2 focus-within:outline-plum",
        !last && "border-b border-line",
        disabled ? "opacity-50" : "cursor-pointer",
      )}
    >
      <Icon name={icon} size={22} className="shrink-0 text-stone" />
      <span className="grow truncate">{label}</span>
      <span
        className={cx(
          "tabular truncate",
          (options ? value : text) ? "font-semibold text-plum" : "text-stone",
        )}
      >
        {display ||
          (options
            ? shown(value, type, options, placeholder)
            : shown(text, type, undefined, placeholder))}
      </span>

      {options && onChange ? (
        <select
          id={id}
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
          className={native}
          aria-label={label}
        >
          {/* Keep an out-of-list value so choosing nothing doesn't change it. */}
          {options.some((o) => o.value === value) || !value ? null : (
            <option value={value}>{value}</option>
          )}
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      ) : onChange ? (
        <input
          id={id}
          type={type ?? "text"}
          value={text}
          placeholder={placeholder}
          disabled={disabled}
          // Date/time inputs fire onChange per digit, so we draft and commit on
          // blur instead of calling onChange straight through on every keystroke.
          onChange={(e) => setDraft(e.target.value)}
          onBlur={commit}
          onKeyDown={(e: KeyboardEvent<HTMLInputElement>) => {
            if (e.key === "Enter") {
              e.preventDefault();
              e.currentTarget.blur();
            }
          }}
          onClick={openPicker}
          className={native}
          aria-label={label}
        />
      ) : null}

      <Icon
        name="right"
        size={16}
        className={cx("shrink-0 text-stone", !onChange && "invisible")}
        aria-hidden="true"
      />
    </label>
  );
}
