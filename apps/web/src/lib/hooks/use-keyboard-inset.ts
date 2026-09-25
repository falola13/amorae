"use client";

import { useEffect } from "react";

/**
 * Publishes the on-screen keyboard's height as `--kb` on <html>.
 *
 * Android Chrome shrinks the page for the keyboard (the viewport's
 * interactive-widget=resizes-content), so there this stays 0. iOS Safari
 * never does: the keyboard slides over a page that keeps its full height, and
 * anything pinned to the bottom — every sheet — ends up underneath it. The
 * visual viewport is the only honest measure of what's still showing.
 */
export function useKeyboardInset() {
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;
    const root = document.documentElement;
    const update = () => {
      const covered = Math.max(0, window.innerHeight - vv.height - vv.offsetTop);
      // Under ~80px is browser chrome settling, not a keyboard.
      root.style.setProperty("--kb", `${covered > 80 ? Math.round(covered) : 0}px`);
    };
    // A focused field inside something scrollable (a tall sheet) should end up
    // in view once the keyboard has taken its space.
    const onFocus = (e: FocusEvent) => {
      const el = e.target as HTMLElement | null;
      if (!el?.matches?.("input, textarea, select, [contenteditable]")) return;
      window.setTimeout(() => el.scrollIntoView({ block: "nearest" }), 300);
    };
    update();
    vv.addEventListener("resize", update);
    vv.addEventListener("scroll", update);
    document.addEventListener("focusin", onFocus);
    return () => {
      vv.removeEventListener("resize", update);
      vv.removeEventListener("scroll", update);
      document.removeEventListener("focusin", onFocus);
      root.style.removeProperty("--kb");
    };
  }, []);
}
