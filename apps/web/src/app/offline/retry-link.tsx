"use client";

import { buttonClassName } from "@/components/ui/button";

// The service worker serves the offline page *at the URL the user asked
// for*, so reloading retries exactly that page. The href is the no-JS
// fallback in case the page's scripts weren't cached.
export function RetryLink() {
  return (
    <a
      href="/dashboard"
      className={buttonClassName("primary")}
      onClick={(event) => {
        event.preventDefault();
        window.location.reload();
      }}
    >
      Try again
    </a>
  );
}
