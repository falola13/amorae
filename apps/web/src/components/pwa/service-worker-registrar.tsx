"use client";

import { useEffect } from "react";
import { watchForUpdates } from "@/lib/pwa/update";

// Production only. In dev, unregisters any worker left behind by a prod run on
// the same origin instead — a live worker would serve stale files over hot reload.
export function ServiceWorkerRegistrar() {
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;

    if (process.env.NODE_ENV !== "production") {
      navigator.serviceWorker
        .getRegistrations()
        .then((registrations) => Promise.all(registrations.map((r) => r.unregister())))
        .catch(() => {});
      return;
    }

    // updateViaCache "none": always revalidate sw.js so a new VERSION reaches users.
    let stop: (() => void) | undefined;
    navigator.serviceWorker
      .register("/sw.js", { scope: "/", updateViaCache: "none" })
      .then((registration) => {
        stop = watchForUpdates(registration);
      })
      .catch((error: unknown) => {
        // Non-fatal: the app works without a worker, just no install/offline page.
        console.warn("Service worker registration failed", error);
      });
    return () => stop?.();
  }, []);

  return null;
}
