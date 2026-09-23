"use client";

import { useEffect, useRef } from "react";

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * What a modal dialog owes the keyboard, in one place: Escape closes it, Tab
 * stays inside it, focus starts on the first control, and on close it goes
 * back to whatever opened it — otherwise a keyboard or screen-reader user is
 * left behind the dialog, or dumped at the top of the page.
 *
 * Returns the ref to put on the dialog element (give it `tabIndex={-1}` so it
 * can hold focus when it has no controls of its own).
 */
export function useDialog(open: boolean, onClose: () => void) {
  const ref = useRef<HTMLDivElement>(null);
  // Kept in a ref so a new inline onClose each render doesn't re-run the
  // effect below, which would steal focus back on every keystroke.
  const close = useRef(onClose);
  useEffect(() => {
    close.current = onClose;
  });

  useEffect(() => {
    if (!open) return;
    const dialog = ref.current;
    const opener = document.activeElement as HTMLElement | null;
    const focusable = () =>
      dialog
        ? [...dialog.querySelectorAll<HTMLElement>(FOCUSABLE)].filter(
            (el) => el.offsetParent !== null,
          )
        : [];

    (focusable()[0] ?? dialog)?.focus();

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        close.current();
        return;
      }
      if (event.key !== "Tab" || !dialog) return;
      const items = focusable();
      if (items.length === 0) {
        event.preventDefault();
        dialog.focus();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      const active = document.activeElement;
      if (!dialog.contains(active)) {
        event.preventDefault();
        (event.shiftKey ? last : first).focus();
      } else if (event.shiftKey && active === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && active === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", onKeyDown, true);
    // The page behind must not scroll under the dialog.
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    return () => {
      document.removeEventListener("keydown", onKeyDown, true);
      document.body.style.overflow = previousOverflow;
      opener?.focus?.();
    };
  }, [open]);

  return ref;
}
