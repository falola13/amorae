"use client";

import { QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { Toaster } from "@/components/layout/toaster";
import { prayerWrites } from "@/features/prayers/writes";
import { settingsWrites } from "@/features/settings/writes";
import { togetherWrites } from "@/features/together/writes";
import { createQueryClient } from "@/lib/query/client";
import { registerWrites, type AnyWriteDef } from "@/lib/query/mutations";

// Every write that can be made offline and sent after a restart. This file
// is the frontend's composition root: the one place that knows every feature,
// so lib/ never has to import one. A new feature adds its writes here.
const resumableWrites: AnyWriteDef[] = [
  ...Object.values(prayerWrites),
  ...Object.values(togetherWrites),
  ...Object.values(settingsWrites),
];

// One QueryClient per browser tab (policies in lib/query/client.ts). Server
// state lives there; client-only state (prayer-mode session, install
// prompt, toasts) lives in the Zustand stores under src/lib/store.
export function Providers({ children }: { children: ReactNode }) {
  const [client] = useState(() => {
    const qc = createQueryClient();
    registerWrites(qc, resumableWrites);
    return qc;
  });
  return (
    <QueryClientProvider client={client}>
      {children}
      <Toaster />
    </QueryClientProvider>
  );
}
