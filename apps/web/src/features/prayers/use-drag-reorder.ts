"use client";

import { useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

import type { PrayerPoint } from "@/lib/api/types";

/**
 * Reorder handlers for the setter's prayer list: a drag handle moves an item
 * up or down a step at a time (pointer, every 60px of travel), or the arrow
 * keys move it one step (keyboard). A pointer drag only updates the
 * on-screen order as it moves and commits once on release, so one drag
 * sends one save — never one per 60px step. A keyboard move has no
 * "release" to batch against, so it commits immediately.
 */
export function useDragReorder(source: PrayerPoint[] | null, onCommit: (next: PrayerPoint[]) => void) {
  const [local, setLocal] = useState<PrayerPoint[] | null>(null);
  const items = local ?? source;
  const drag = useRef<{ from: number; y: number } | null>(null);
  const dirty = useRef<PrayerPoint[] | null>(null);

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
    drag.current = { from: i, y: e.clientY };
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
  };
  const onPointerMove = (e: ReactPointerEvent) => {
    if (!drag.current || !items) return;
    const dy = e.clientY - drag.current.y;
    if (Math.abs(dy) > 60) {
      const to = drag.current.from + Math.sign(dy);
      const next = reordered(items, drag.current.from, to);
      if (next) { setLocal(next); dirty.current = next; drag.current = { from: to, y: e.clientY }; }
    }
  };
  const endDrag = () => {
    drag.current = null;
    if (dirty.current) { onCommit(dirty.current); dirty.current = null; }
  };

  return {
    items,
    moveByKey,
    listProps: { onPointerMove, onPointerUp: endDrag, onPointerCancel: endDrag },
    handleProps: (i: number) => ({ onPointerDown: (e: ReactPointerEvent) => onPointerDown(i, e) }),
  };
}
