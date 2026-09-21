import Link from "next/link";
import type { ReactNode } from "react";

import { Mark } from "@/components/brand/logo";

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <main className="flex flex-1 flex-col items-center justify-center gap-8 bg-bg px-6 py-12">
      <Link href="/" aria-label="Amorae home" className="text-accent">
        <Mark size={44} />
      </Link>
      <div className="w-full max-w-sm">{children}</div>
    </main>
  );
}
