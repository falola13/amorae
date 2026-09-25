"use client";

import { QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { Toaster } from "@/components/layout/toaster";
import { prayerWrites } from "@/features/prayers/writes";
import { settingsWrites } from "@/features/settings/writes";
import { togetherWrites } from "@/features/together/writes";
import { useKeyboardInset } from "@/lib/hooks/use-keyboard-inset";
import { createQueryClient } from "@/lib/query/client";
import { registerWrites, type AnyWriteDef } from "@/lib/query/mutations";

// Composition root: the one place that knows every feature's resumable
// writes, so lib/ never has to import one. A new feature adds its writes here.
const resumableWrites: AnyWriteDef[] = [
  ...Object.values(prayerWrites),
  ...Object.values(togetherWrites),
  ...Object.values(settingsWrites),
];

// One QueryClient per browser tab. Client-only state (prayer-mode session,
// install prompt, toasts) lives in the Zustand stores under src/lib/store instead.
export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(() => {
    const qc = createQueryClient();
    registerWrites(qc, resumableWrites);
    return qc;
  });
  useKeyboardInset();
  return (
    <QueryClientProvider client={client}>
      {children}
      <Toaster />
    </QueryClientProvider>
  );
}
