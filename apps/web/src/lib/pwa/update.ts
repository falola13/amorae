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
  // `fresh`: the app was just opened or brought back to the front, so
  // nothing is mid-way and reloading onto the new version costs nobody
  // anything. Later checks, while it is in use, only raise the banner.
  const check = async (fresh: boolean) => {
    try {
      const res = await fetch("/api/version", { cache: "no-store" });
      if (!res.ok) return;
      const { build } = (await res.json()) as { build: string | null };
      if (!build || build === mine) return;
      if (fresh && reloadOnce(build)) return;
      useUpdate.getState().markReady();
    } catch {
      // Offline or mid-deploy: the next look will do.
    }
  };
  const onVisible = () => {
    if (document.visibilityState === "visible") void check(true);
  };
  void check(true);
  document.addEventListener("visibilitychange", onVisible);
  const timer = window.setInterval(() => void check(false), EVERY_MS);
  return () => {
    document.removeEventListener("visibilitychange", onVisible);
    window.clearInterval(timer);
  };
}

// Screens where someone may be writing; reloading would throw it away.
const WRITING = /\/(new|edit|set|letter|journal|appreciation|mode)(\/|$|\?)/;

/**
 * Reloads onto `build`, at most once per build per tab — if the server and
 * this page somehow keep disagreeing, the banner takes over rather than a
 * loop. Never on a screen where something may be half written.
 */
function reloadOnce(build: string): boolean {
  if (WRITING.test(window.location.pathname)) return false;
  const key = `amorae.reloaded-for.${build}`;
  try {
    if (window.sessionStorage.getItem(key)) return false;
    window.sessionStorage.setItem(key, "1");
  } catch {
    return false;
  }
  window.location.reload();
  return true;
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
