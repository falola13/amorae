"use client";

import { useEffect } from "react";

import { Toast } from "@/components/ui/kit";
import { useToast } from "@/lib/store/toast";

const VISIBLE_MS = 4000;

/** Renders the current toast (lib/store/toast.ts) and clears it after a few seconds. */
export function Toaster() {
  const { message, seq, dismiss } = useToast();

  useEffect(() => {
    if (!message) return;
    const timer = setTimeout(dismiss, VISIBLE_MS);
    return () => clearTimeout(timer);
  }, [message, seq, dismiss]);

  return message ? <Toast message={message} action="Dismiss" onAction={dismiss} /> : null;
}
