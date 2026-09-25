"use client";

import { create } from "zustand";

// A new service worker takes over at once (sw.js skipWaiting), but a page
// already open keeps running the JavaScript it loaded — for days, in an
// installed app nobody closes. This flag is how the page learns it is behind.

interface UpdateState {
  ready: boolean;
  markReady: () => void;
}

export const useUpdate = create<UpdateState>()((set) => ({
  ready: false,
  markReady: () => set({ ready: true }),
}));

/** Listens for a new worker taking over, and asks for one whenever the app comes back to the front. */
export function watchForUpdates(registration: ServiceWorkerRegistration): () => void {
  // No controller yet means this is the first install, not an update.
  const hadController = !!navigator.serviceWorker.controller;
  const onControllerChange = () => {
    if (hadController) useUpdate.getState().markReady();
  };
  // Browsers only re-check sw.js on a real navigation, and an app that routes
  // on the client rarely makes one — so check on return to the foreground.
  const onVisible = () => {
    if (document.visibilityState === "visible") registration.update().catch(() => {});
  };
  navigator.serviceWorker.addEventListener("controllerchange", onControllerChange);
  document.addEventListener("visibilitychange", onVisible);
  return () => {
    navigator.serviceWorker.removeEventListener("controllerchange", onControllerChange);
    document.removeEventListener("visibilitychange", onVisible);
  };
}
