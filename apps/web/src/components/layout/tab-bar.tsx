"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import { Icon } from "@/components/icons";
import { cx } from "@/components/ui/kit";
import { NAV } from "./nav-items";
import { Ribbon } from "./ribbon";

/** Hides while the on-screen keyboard is up (visualViewport shrinks). */
function useKeyboardOpen() {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;
    const check = () => setOpen(window.innerHeight - vv.height > 150);
    vv.addEventListener("resize", check);
    return () => vv.removeEventListener("resize", check);
  }, []);
  return open;
}

export function TabBar() {
  const path = usePathname();
  const hidden = useKeyboardOpen();
  if (hidden) return null;
  return (
    // The foot of the page: the same stock and grain as the page, one hairline
    // above, and nothing floating, filled or blurred.
    <nav
      aria-label="Primary"
      className="grain sticky bottom-0 z-30 flex shrink-0 border-t border-line bg-bg px-1 md:hidden"
      style={{ paddingBottom: "max(var(--safe-bottom), 16px)" }}
    >
      {NAV.map((t) => {
        const on = t.match(path);
        return (
          <Link
            key={t.href}
            href={t.href}
            aria-current={on ? "page" : undefined}
            className={cx(
              "press relative flex h-16 min-w-11 grow basis-0 flex-col items-center justify-start gap-1 pt-5 text-micro tracking-[0.01em] no-underline",
              on ? "font-[750] text-ink" : "font-semibold text-stone",
            )}
          >
            {on ? <Ribbon dir="down" /> : null}
            <Icon name={t.icon} size={22} strokeWidth={on ? 1.9 : 1.5} />
            <span>{t.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
