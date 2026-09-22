import type { Metadata } from "next";

import { Mark } from "@/components/icons";
import { RetryLink } from "./retry-link";

export const metadata: Metadata = { title: "Offline", robots: { index: false } };

// Precached by public/sw.js and shown when a page can't be reached at all.
// Static and free of user data: the worker stores one copy for everyone.
// Bump VERSION in public/sw.js when this changes.
export default function OfflinePage() {
  return (
    <main className="mx-auto flex min-h-dvh w-full max-w-[520px] flex-col items-center justify-center gap-6 px-6 text-center">
      <Mark size={48} className="text-plum" />
      <div className="flex flex-col gap-2">
        <h1 className="m-0 text-[24px] font-semibold leading-tight tracking-[-0.02em]">You&rsquo;re offline</h1>
        <p className="m-0 max-w-xs text-body text-stone">This page needs a connection. Anything you opened earlier is still here once you go back.</p>
      </div>
      <RetryLink />
    </main>
  );
}
