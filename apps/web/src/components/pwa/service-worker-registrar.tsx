"use client";

import { useEffect } from "react";

// Registers /sw.js in production builds only. In development it does the
// opposite and removes any worker left behind by a production run on the
// same origin (docker compose and `npm run dev` both use localhost:3000),
// because a live worker would keep serving old static files over hot reload.
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

    // updateViaCache "none" makes the browser always revalidate sw.js with
    // the server, so a new VERSION reaches users on their next visit.
    navigator.serviceWorker
      .register("/sw.js", { scope: "/", updateViaCache: "none" })
      .catch((error: unknown) => {
        // The app works without a worker (no install, no offline page), so
        // this is logged rather than surfaced to the user.
        console.warn("Service worker registration failed", error);
      });
  }, []);

  return null;
}
