"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { Icon, Mark } from "@/components/icons";
import { cx } from "@/components/ui/kit";
import { NAV } from "./nav-items";
import { routes } from "@/lib/routes";

/**
 * Primary navigation on a tablet or desktop, where a bottom bar would be a long
 * reach from the hands and waste the width. A rail of icons from `md`, widening
 * to labelled rows from `lg`.
 *
 * Only one of this and the TabBar is ever displayed: each is `display: none` at
 * the other's widths, which also keeps it out of the accessibility tree, so
 * there is never a second "Primary" navigation landmark.
 */
export function SideNav() {
  const path = usePathname();
  return (
    <nav
      aria-label="Primary"
      className="sticky top-0 hidden h-[100dvh] shrink-0 flex-col gap-1 border-r border-line px-3 md:flex md:w-[88px] lg:w-[236px] lg:px-4"
      style={{ paddingTop: "calc(var(--safe-top) + 20px)", paddingBottom: "var(--safe-bottom)" }}
    >
      <Link
        href={routes.home}
        aria-label="Amorae, home"
        className="press mb-4 flex h-11 items-center gap-2.5 rounded-btn text-ink no-underline md:justify-center lg:justify-start lg:px-2"
      >
        <Mark size={26} className="text-plum" />
        <span className="hidden text-[19px] font-semibold tracking-[-0.02em] lg:inline">
          Amorae
        </span>
      </Link>
      {NAV.map((item) => {
        const on = item.match(path);
        return (
          <Link
            key={item.href}
            href={item.href}
            aria-current={on ? "page" : undefined}
            className={cx(
              "press flex items-center rounded-btn no-underline",
              // Rail: icon over label, centred. Wide: one row, label beside icon.
              "md:h-[58px] md:flex-col md:justify-center md:gap-1 md:text-[11px] md:tracking-[0.02em]",
              "lg:h-12 lg:flex-row lg:justify-start lg:gap-3 lg:px-3 lg:text-[16px] lg:tracking-normal",
              on
                ? "bg-plum-tint font-bold text-plum lg:font-semibold"
                : "font-semibold text-stone hover:bg-paper hover:text-ink",
            )}
          >
            <Icon name={item.icon} size={22} strokeWidth={on ? 1.7 : 1.5} />
            <span>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
