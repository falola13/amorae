"use client";

import { useState, useSyncExternalStore } from "react";

import { http } from "@/lib/api/http";

// "unsupported" covers both "this browser cannot" and "this build has no
// service worker to subscribe through", because to the person holding the
// phone those are the same sentence: notifications cannot reach you here.
type PushState = "unsupported" | "default" | "granted" | "denied";
const current = (): PushState =>
  !("Notification" in window) || !("serviceWorker" in navigator)
    ? "unsupported"
    : Notification.permission;

function urlBase64ToUint8Array(base64: string) {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const raw = atob((base64 + padding).replace(/-/g, "+").replace(/_/g, "/"));
  return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}

// Ask only after the permission screen has explained the value (spec 19.3).
export function usePush() {
  const initial = useSyncExternalStore(
    () => () => {},
    current,
    () => "default" as PushState,
  );
  const [override, setState] = useState<PushState | null>(null);
  const state = override ?? initial;
  const request = async (): Promise<PushState> => {
    if (!("Notification" in window)) return "unsupported";
    const p = await Notification.requestPermission();
    setState(p);
    if (p !== "granted") return p;
    const key = process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY;
    try {
      // navigator.serviceWorker.ready never settles when there is no worker
      // to become ready — it waits, forever, without throwing. A development
      // build is exactly that case: ServiceWorkerRegistrar unregisters
      // workers there so hot reload is not served stale files. Awaiting it
      // meant this function simply never returned, the button stayed busy,
      // and nobody was ever subscribed. Asking for the registration first
      // turns that silence into an answer.
      const existing = await navigator.serviceWorker.getRegistration();
      if (!existing) return "unsupported";

      const reg = await navigator.serviceWorker.ready;
      const sub =
        (await reg.pushManager.getSubscription()) ??
        (key
          ? await reg.pushManager.subscribe({
              userVisibleOnly: true,
              applicationServerKey: urlBase64ToUint8Array(key),
            })
          : null);
      if (sub) await http.post("/notifications/subscribe", sub.toJSON());
    } catch {
      /* subscription is best effort; the preference is still saved */
    }
    return p;
  };
  return { state, request };
}
