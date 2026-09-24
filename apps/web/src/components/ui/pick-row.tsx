"use client";

import { useId, useState, type KeyboardEvent, type MouseEvent } from "react";
import { Icon, type IconName } from "@/components/icons";
import { cx } from "@/components/ui/kit";
import { longDate, time12 } from "@/lib/dates";

/**
 * A row on a compose or detail screen holding one small value: a date, a
 * time, a choice from a list, or a short piece of text.
 *
 * The control is always a real native one — <input type="date">, <input
 * type="time">, <select> — because that is what gives each platform the
 * picker its own users already know: the wheel on iOS, the dialog on Android,
 * the dropdown on desktop. Nothing here reimplements a calendar, so there is
 * nothing to keep working as those four change, and the keyboard and
 * screen-reader behaviour is already right.
 *
 * It sits invisibly over the row so the value can be shown in the app's own
 * words — "24 September", not 09/24/2026, and "Add a time" rather than the
 * browser's "--:--". Rendering the native field visibly instead was tried and
 * reverted: it is more robust and looks considerably cheaper, and it puts the
 * month before the day for a product used in Lagos.
 *
 * What the invisible field cost before, and what is fixed here:
 *
 *   - It looked like text. Nothing said a row could be tapped, so it read as
 *     dead. Every row now carries a chevron and lights its icon on focus.
 *   - Keyboard users had no idea where they were: the ring belonged to an
 *     invisible box. The row shows it now, through focus-within.
 *   - On desktop, focus alone does not open a date dropdown, so the picker
 *     had to be asked for. openPicker does that on every click rather than
 *     only the first — a second click on an already-focused field opens
 *     nothing by itself, which is exactly "it opens once and then won't".
 *
 * On iOS and Android none of that last part applies: focusing a date or time
 * field opens the native picker by itself, which is why the field stays
 * native rather than becoming a sheet of our own.
 */
function openPicker(event: MouseEvent<HTMLInputElement>) {
  const input = event.currentTarget;
  if (input.disabled || (input.type !== "date" && input.type !== "time")) return;
  try {
    input.showPicker();
  } catch {
    /* Older browser, or it declined because this was not a real gesture. The
       click still focuses the field, which is what phones act on anyway. */
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
  /**
   * What to show instead of the value itself, when the value is not what a
   * person should read — a target of 500000 shown as ₦500,000. Dates and
   * times never need it: this row formats those.
   */
  display?: string;
  type?: PickType;
  options?: Options;
  onChange?: (v: string) => void;
  placeholder?: string;
  disabled?: boolean;
  last?: boolean;
}) {
  const id = useId();
  // Null while nobody is typing, so the row follows `value`; a string once
  // they are, so their own keystrokes are what they see.
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
          {/* A value we were given but do not offer is kept, so choosing
              nothing changes nothing. */}
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
          // React's onChange on an input is the DOM *input* event, which a
          // date or time field fires on every digit — so "19:30" was four
          // separate values, and on a row that saves as you go, four separate
          // requests, three of them for times nobody meant. Every kind of
          // field now holds a draft and reports it when the person is done
          // with it: on blur, which is also when a phone's picker closes.
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
