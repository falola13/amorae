"use client";

import { create } from "zustand";

// A store, not React context, so non-React code (the query client's error handler) can call notify().

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
