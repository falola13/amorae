"use client";

import { buttonClass } from "@/components/ui/kit";

// The service worker serves this page at the URL the user requested, so reload retries that exact page.
export function RetryLink() {
  return (
    <button
      type="button"
      className={buttonClass("primary")}
      onClick={() => window.location.reload()}
    >
      Try again
    </button>
  );
}
