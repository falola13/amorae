"use client";

import { create } from "zustand";

// One short message at a time, shown by <Toaster /> above the tab bar. Kept
// in a store (not React context) so non-React code, like the query client's
// global mutation error handler, can raise one with notify().

interface ToastState {
  message: string | null;
  seq: number; // bumps on every show, so an identical message still restarts the timer
  show: (message: string) => void;
  dismiss: () => void;
}

export const useToast = create<ToastState>()((set) => ({
  message: null,
  seq: 0,
  show: (message) => set((s) => ({ message, seq: s.seq + 1 })),
  dismiss: () => set({ message: null }),
}));

export const notify = (message: string) => useToast.getState().show(message);
