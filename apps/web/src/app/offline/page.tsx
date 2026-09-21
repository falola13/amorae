import type { Metadata } from "next";

import { Mark } from "@/components/brand/logo";
import { RetryLink } from "./retry-link";

export const metadata: Metadata = {
  title: "Offline",
  robots: { index: false },
};

// Precached by public/sw.js and shown whenever a page can't be reached. It
// must stay static and free of user data, because the worker stores one
// copy for everyone. If you change it, bump VERSION in public/sw.js so
// installed apps pick up the new copy.
export default function OfflinePage() {
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-6 px-6 text-center">
      <Mark size={48} className="text-accent" />
      <div className="flex flex-col gap-2">
        <h1 className="font-display text-2xl text-fg">You&rsquo;re offline</h1>
        <p className="max-w-xs text-fg-muted">
          Amorae needs a connection to load this page. Check your signal and try again.
        </p>
      </div>
      <RetryLink />
    </main>
  );
}
