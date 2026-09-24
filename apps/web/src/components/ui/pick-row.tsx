"use client";

import { useId, useState, type KeyboardEvent, type MouseEvent } from "react";
import { Icon, type IconName } from "@/components/icons";
import { cx } from "@/components/ui/kit";

/**
 * Opens the native picker on every click, not just the first.
 *
 * The input is invisible and fills the row, so a tap anywhere lands on it.
 * The browser opens the picker when that first click focuses the input — and
 * then never again, because the input is already focused and a click on a
 * date field's text area does not reopen it. Asking for it explicitly is the
 * only thing that behaves the same way twice.
 *
 * showPicker needs a real user gesture, which a click handler is, and throws
 * for an input type that has nothing to show — so it is guarded and the throw
 * is swallowed rather than becoming a broken row.
 */
function openPicker(event: MouseEvent<HTMLInputElement>) {
  const input = event.currentTarget;
  if (input.type !== "date" && input.type !== "time") return;
  try {
    input.showPicker();
  } catch {
    /* Older browser, or it declined: the click still focuses the field, which
       is what happened before this existed. */
  }
}

/**
 * A row on a compose screen that opens a native picker (date, time), offers a
 * fixed set of choices, or holds a short value.
 *
 * With `options` the overlay is a native <select>, so the choice arrives as a
 * wheel on a phone and a menu on a desktop rather than a text field nobody can
 * see. A value that is not in the list is kept and offered as its own option:
 * an event written before the list existed must not silently become whatever
 * happens to be first.
 *
 * A text row holds a draft while it is being typed into and reports it when
 * the person finishes — on blur, or on Enter. A date, a time and a choice are
 * single decisions and report straight away. The difference matters where the
 * row saves rather than filling in a form: a per-keystroke onChange there is a
 * request per letter, and each one comes back and fights what is being typed.
 */
export function PickRow({
  icon,
  label,
  value,
  empty,
  type,
  options,
  onChange,
  placeholder,
  last,
}: {
  icon: IconName;
  label: string;
  value: string;
  empty?: string;
  type?: "date" | "time" | "text";
  options?: readonly { value: string; label: string }[];
  onChange?: (v: string) => void;
  placeholder?: string;
  last?: boolean;
}) {
  const id = useId();
  const chosen = options?.find((o) => o.value === value);
  // Null while nobody is typing, so the row follows `value`; a string once
  // they are, so their own keystrokes are what they see.
  const [draft, setDraft] = useState<string | null>(null);
  const text = draft ?? value;
  const commit = () => {
    if (draft !== null && draft !== value) onChange?.(draft);
    setDraft(null);
  };
  const shown = options
    ? (chosen?.label ?? value ?? placeholder ?? "")
    : text && type !== "date" && type !== "time"
      ? text
      : empty || placeholder || "";
  return (
    <label
      htmlFor={id}
      className={cx(
        "relative flex h-[54px] w-full cursor-pointer items-center gap-3.5 text-[16px] font-medium text-ink",
        !last && "border-b border-line",
      )}
    >
      <Icon name={icon} size={22} className="text-stone" />
      <span className="grow">{label}</span>
      <span className={cx("tabular", value ? "font-semibold text-plum" : "text-stone")}>
        {shown}
      </span>
      {options && onChange ? (
        <select
          id={id}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
          aria-label={label}
        >
          {chosen || !value ? null : <option value={value}>{value}</option>}
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
          value={type === "date" || type === "time" ? value : text}
          placeholder={placeholder}
          onChange={(e) =>
            type === "date" || type === "time" ? onChange(e.target.value) : setDraft(e.target.value)
          }
          onBlur={commit}
          onKeyDown={(e: KeyboardEvent<HTMLInputElement>) => {
            if (e.key === "Enter") {
              e.preventDefault();
              e.currentTarget.blur();
            }
          }}
          onClick={openPicker}
          className={cx(
            "absolute inset-0 h-full w-full cursor-pointer opacity-0",
            type === "text" && "opacity-0",
          )}
          aria-label={label}
        />
      ) : null}
    </label>
  );
}
