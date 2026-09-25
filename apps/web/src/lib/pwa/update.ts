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

const EVERY_MS = 15 * 60 * 1000;

/**
 * Asks the server which deploy it is serving, on return to the foreground and
 * every so often while open. This is the signal that actually tracks deploys:
 * sw.js is unchanged by most of them, so no new worker ever arrives.
 */
export function watchBuild(): () => void {
  const mine = process.env.NEXT_PUBLIC_BUILD_ID;
  if (!mine) return () => {};
  const check = async () => {
    try {
      const res = await fetch("/api/version", { cache: "no-store" });
      if (!res.ok) return;
      const { build } = (await res.json()) as { build: string | null };
      if (build && build !== mine) useUpdate.getState().markReady();
    } catch {
      // Offline or mid-deploy: the next look will do.
    }
  };
  const onVisible = () => {
    if (document.visibilityState === "visible") void check();
  };
  document.addEventListener("visibilitychange", onVisible);
  const timer = window.setInterval(check, EVERY_MS);
  return () => {
    document.removeEventListener("visibilitychange", onVisible);
    window.clearInterval(timer);
  };
}

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
