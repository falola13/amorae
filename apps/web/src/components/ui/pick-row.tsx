"use client";

import { useId, type MouseEvent } from "react";
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

/** A row on a compose screen that opens a native picker (date, time) or holds a short value. */
export function PickRow({
  icon,
  label,
  value,
  empty,
  type,
  onChange,
  placeholder,
  last,
}: {
  icon: IconName;
  label: string;
  value: string;
  empty?: string;
  type?: "date" | "time" | "text";
  onChange?: (v: string) => void;
  placeholder?: string;
  last?: boolean;
}) {
  const id = useId();
  const shown = value && type !== "date" && type !== "time" ? value : empty || placeholder || "";
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
      {onChange ? (
        <input
          id={id}
          type={type ?? "text"}
          value={value}
          placeholder={placeholder}
          onChange={(e) => onChange(e.target.value)}
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
