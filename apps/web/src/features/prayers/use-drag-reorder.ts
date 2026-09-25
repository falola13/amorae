"use client";

import { useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";

import type { PrayerPoint } from "@/lib/api/types";

/** The row being held, and how far it has been dragged since its last swap. */
export type Held = { index: number; offset: number };

/**
 * Reorder handlers for the setter's prayer list.
 *
 * The first version of this worked and looked broken, which is worse than
 * broken. It moved a row only after 60px of travel — on a 73px row, most of
 * a row's height — and drew nothing at all in the meantime. So the handle
 * took your finger, gave no sign it had, and then either jumped or did not.
 * Everybody who tried it concluded the grip was decorative.
 *
 * Two changes, both about the middle of the gesture rather than its ends.
 * The row now follows the finger from the first pixel, so the grab is visible
 * before anything has moved. And it swaps at half a row rather than a fixed
 * 60px, measured from the row itself, so the swap happens exactly when the
 * row has covered half its neighbour — which is where the eye expects it.
 *
 * A pointer drag still commits once, on release: one drag is one save, never
 * one per step. A keyboard move has no release to batch against, so it
 * commits immediately.
 */
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
    // currentTarget, not target: the press usually lands on the icon's path,
    // and capturing there worked only by accident of bubbling.
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
        // The row has taken its neighbour's place, so the finger is level
        // with it again: new baseline, no offset.
        drag.current = { ...d, from: to, y: e.clientY };
        setHeld({ index: to, offset: 0 });
        return;
      }
    }
    // At the ends of the list there is nowhere to swap to, and the row
    // follows anyway — the resistance is what says "this is as far as it
    // goes", which a row that simply ignored you would not.
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
