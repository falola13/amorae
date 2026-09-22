"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

// Small pieces of client-only state that outlive a screen:
// - the prayer-mode session (which prayer, the quiet timer)
// - whether install guidance was dismissed (spec 19.2: remember it)
// - the browser's deferred install prompt (Android)
interface UIState {
  prayerIndex: number | null;
  timerStartedAt: number | null;
  installDismissed: boolean;
  installPrompt: (() => Promise<void>) | null;
  setPrayerIndex: (i: number | null) => void;
  toggleTimer: () => void;
  dismissInstall: () => void;
  setInstallPrompt: (p: (() => Promise<void>) | null) => void;
}

export const useUI = create<UIState>()(
  persist(
    (set, get) => ({
      prayerIndex: null,
      timerStartedAt: null,
      installDismissed: false,
      installPrompt: null,
      setPrayerIndex: (prayerIndex) => set({ prayerIndex }),
      toggleTimer: () => set({ timerStartedAt: get().timerStartedAt === null ? Date.now() : null }),
      dismissInstall: () => set({ installDismissed: true }),
      setInstallPrompt: (installPrompt) => set({ installPrompt }),
    }),
    { name: "amorae:ui", partialize: (s) => ({ installDismissed: s.installDismissed }) },
  ),
);
