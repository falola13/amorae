import Link from "next/link";
import type { ReactNode } from "react";

import { Lockup } from "@/components/brand/logo";
import { LogoutButton } from "@/features/auth/components/logout-button";

export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-1 flex-col bg-bg">
      <header className="border-b border-border bg-bg-elevated">
        {/* Tight vertical padding: the lockup SVG already includes its clear space. */}
        <div className="mx-auto flex max-w-3xl items-center justify-between px-3 py-1 pr-6">
          <Link href="/dashboard" aria-label="Amorae home">
            <Lockup height={56} />
          </Link>
          <LogoutButton />
        </div>
      </header>
      <main className="mx-auto w-full max-w-3xl px-6 py-10">{children}</main>
    </div>
  );
}
