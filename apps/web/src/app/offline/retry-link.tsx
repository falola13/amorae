"use client";

import { buttonClass } from "@/components/ui/kit";

// The service worker serves the offline page *at the URL the user asked
// for*, so reloading retries exactly that page.
export function RetryLink() {
  return (
    <button type="button" className={buttonClass("primary")} onClick={() => window.location.reload()}>
      Try again
    </button>
  );
}
