"use client";

import { useEffect, useSyncExternalStore } from "react";

import { useUI } from "@/lib/store/ui";

const noop = () => () => {};
const isStandalone = () => window.matchMedia("(display-mode: standalone)").matches || (navigator as unknown as { standalone?: boolean }).standalone === true;
const isIOS = () => /iphone|ipad|ipod/i.test(navigator.userAgent);

// Standalone detection and the Android install prompt. iOS has no prompt API,
// so the install screen shows the Add to Home Screen steps there instead.
export function useInstall() {
  const standalone = useSyncExternalStore(noop, isStandalone, () => false);
  const ios = useSyncExternalStore(noop, isIOS, () => false);
  const prompt = useUI((s) => s.installPrompt);
  const setPrompt = useUI((s) => s.setInstallPrompt);
  useEffect(() => {
    const handler = (raw: globalThis.Event) => {
      const e = raw as globalThis.Event & { prompt?: () => Promise<void> };
      e.preventDefault();
      setPrompt(async () => { await e.prompt?.(); });
    };
    window.addEventListener("beforeinstallprompt", handler);
    return () => window.removeEventListener("beforeinstallprompt", handler);
  }, [setPrompt]);
  return { standalone, ios, prompt };
}
