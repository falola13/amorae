"use client";

import { buttonClassName } from "@/components/ui/button";

interface ErrorPageProps {
  // `error` (with its `digest`) is deliberately unused in the UI — never
  // render error.message to the user; it may contain internal details.
  // Server-side logging of the real error happens where it's thrown.
  error: Error & { digest?: string };
  reset: () => void;
}

export default function ErrorPage({ reset }: ErrorPageProps) {
  return (
    <main className="mx-auto flex w-full max-w-md flex-1 flex-col items-center justify-center gap-4 px-6 text-center">
      <h1 className="font-display text-3xl text-fg">Something went wrong</h1>
      <p className="text-fg-muted">We hit a snag loading this page. Please try again.</p>
      <button type="button" onClick={reset} className={buttonClassName("primary")}>
        Try again
      </button>
    </main>
  );
}
