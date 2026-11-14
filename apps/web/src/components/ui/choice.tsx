"use client";

import { type ReactNode } from "react";
import { Icon } from "@/components/icons";
import { cx } from "./cx";

/* ---------- Choosing one of a few ---------- */

type Option<T> = { value: T; label: ReactNode };

/**
 * Underlined tabs, for switching what a list shows. Not pills: a filter is a
 * place in the page, not a button on it. The current tab is ink at a heavier
 * weight as well as plum-underlined, so it never rests on colour alone.
 *
 * The hairline runs the full width of the column; pass the negative gutter in
 * `className` (e.g. `-mx-6 px-3.5`) to let it reach the screen's edges.
 */
export function Tabs<T extends string>({
  label,
  options,
  value,
  onChange,
  className,
}: {
  label: string;
  options: readonly Option<T>[];
  value: T;
  onChange: (v: T) => void;
  className?: string;
}) {
  return (
    <div
      role="tablist"
      aria-label={label}
      className={cx("flex gap-1 overflow-x-auto border-b border-line", className)}
    >
      {options.map((o) => {
        const on = o.value === value;
        return (
          <button
            key={o.value}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => onChange(o.value)}
            className={cx(
              "press relative inline-flex h-11 shrink-0 items-center px-2.5 text-[15px]",
              on ? "font-[750] text-ink" : "font-semibold text-stone",
            )}
          >
            {o.label}
            {on ? (
              <span aria-hidden="true" className="absolute inset-x-2.5 -bottom-px h-0.5 bg-plum" />
            ) : null}
          </button>
        );
      })}
    </div>
  );
}

/**
 * A segmented choice (a radiogroup), for picking one of two to six short
 * options in a form. Every segment is 44px tall. The chosen one is filled
 * with ink and carries a check, so it reads without colour.
 *
 * `wrap` lets a longer set (the journal's five kinds) flow onto a second line
 * with segments sized to their words, instead of squeezing them equal.
 */
export function Segmented<T extends string | number>({
  label,
  options,
  value,
  onChange,
  disabled,
  wrap,
  className,
}: {
  label: string;
  options: readonly Option<T>[];
  value: T;
  onChange: (v: T) => void;
  disabled?: boolean;
  wrap?: boolean;
  className?: string;
}) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      className={cx(
        "flex gap-[3px] rounded-btn border border-edge p-[3px]",
        wrap && "flex-wrap",
        disabled && "opacity-55",
        className,
      )}
    >
      {options.map((o) => {
        const on = o.value === value;
        return (
          <button
            key={o.value}
            type="button"
            role="radio"
            aria-checked={on}
            disabled={disabled}
            onClick={() => onChange(o.value)}
            className={cx(
              "press flex h-11 min-w-11 grow items-center justify-center gap-1.5 whitespace-nowrap rounded-[7px] px-3 text-[15px]",
              wrap ? "basis-auto" : "basis-0",
              on ? "bg-ink font-bold text-bg" : "font-semibold text-ink",
            )}
          >
            {on ? <Icon name="check" size={16} strokeWidth={2.2} /> : null}
            {o.label}
          </button>
        );
      })}
    </div>
  );
}
