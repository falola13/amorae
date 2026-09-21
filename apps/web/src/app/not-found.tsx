import Link from "next/link";

import { buttonClassName } from "@/components/ui/button";

export default function NotFound() {
  return (
    <main className="mx-auto flex w-full max-w-md flex-1 flex-col items-center justify-center gap-4 px-6 text-center">
      <h1 className="font-display text-3xl text-fg">Page not found</h1>
      <p className="text-fg-muted">The page you&apos;re looking for doesn&apos;t exist or has moved.</p>
      <Link href="/" className={buttonClassName("primary")}>
        Back home
      </Link>
    </main>
  );
}
