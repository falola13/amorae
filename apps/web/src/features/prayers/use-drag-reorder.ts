"use client";

import { useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

import type { PrayerPoint } from "@/lib/api/types";

/** The row being held, and how far it has been dragged since its last swap. */
export type Held = { index: number; offset: number };

// Reorder handlers for the setter's prayer list. Swaps at half a row of travel; a pointer drag
// commits once on release, a keyboard move commits immediately.
export function useDragReorder(
  source: PrayerPoint[] | null,
  onCommit: (next: PrayerPoint[]) => void,
) {
  const [local, setLocal] = useState<PrayerPoint[] | null>(null);
  const items = local ?? source;
  const drag = useRef<{ from: number; y: number; step: number } | null>(null);
  const dirty = useRef<PrayerPoint[] | null>(null);
  const [held, setHeld] = useState<Held | null>(null);

  const reordered = (list: PrayerPoint[], from: number, to: number) => {
    if (to < 0 || to >= list.length || from === to) return null;
    const next = [...list];
    const [x] = next.splice(from, 1);
    next.splice(to, 0, x);
    return next;
  };

  const moveByKey = (from: number, to: number) => {
    if (!items) return;
    const next = reordered(items, from, to);
    if (!next) return;
    setLocal(next);
    onCommit(next);
  };

  const onPointerDown = (i: number, e: ReactPointerEvent) => {
    // currentTarget, not target — the press usually lands on the icon's inner path.
    const handle = e.currentTarget as HTMLElement;
    const step = handle.closest("li")?.getBoundingClientRect().height ?? 72;
    drag.current = { from: i, y: e.clientY, step };
    setHeld({ index: i, offset: 0 });
    handle.setPointerCapture(e.pointerId);
  };

  const onPointerMove = (e: ReactPointerEvent) => {
    const d = drag.current;
    if (!d || !items) return;
    const dy = e.clientY - d.y;

    if (Math.abs(dy) > d.step / 2) {
      const to = d.from + Math.sign(dy);
      const next = reordered(items, d.from, to);
      if (next) {
        setLocal(next);
        dirty.current = next;
        // Row took its neighbour's place; finger is level again, so reset the baseline.
        drag.current = { ...d, from: to, y: e.clientY };
        setHeld({ index: to, offset: 0 });
        return;
      }
    }
    // At list ends there's nowhere to swap; the row still follows, showing resistance rather than nothing.
    setHeld({ index: d.from, offset: dy });
  };

  const endDrag = () => {
    drag.current = null;
    setHeld(null);
    if (dirty.current) {
      onCommit(dirty.current);
      dirty.current = null;
    }
  };

  return {
    items,
    held,
    moveByKey,
    listProps: { onPointerMove, onPointerUp: endDrag, onPointerCancel: endDrag },
    handleProps: (i: number) => ({ onPointerDown: (e: ReactPointerEvent) => onPointerDown(i, e) }),
  };
}
