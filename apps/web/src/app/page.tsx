import Link from "next/link";

import { Lockup } from "@/components/brand/logo";
import { buttonClassName } from "@/components/ui/button";
import { brand } from "@/lib/brand";

export default function HomePage() {
  return (
    <main className="mx-auto flex w-full max-w-xl flex-1 flex-col items-center justify-center gap-8 px-6 py-12 text-center">
      <h1 className="sr-only">{brand.name}</h1>
      {/* The lockup carries the brand's clear space (half the mark's width
          on every side) inside the SVG, so it needs no extra margin. */}
      <Lockup variant="stacked" height={200} />
      <div className="-mt-6 flex flex-col gap-2">
        <p className="font-display text-xl font-medium text-fg">{brand.tagline}</p>
        <p className="text-fg-muted">{brand.description}</p>
      </div>
      <div className="flex flex-wrap items-center justify-center gap-3">
        <Link href="/register" className={buttonClassName("primary")}>
          Get started
        </Link>
        <Link href="/login" className={buttonClassName("secondary")}>
          Sign in
        </Link>
      </div>
    </main>
  );
}
