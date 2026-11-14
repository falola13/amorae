"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { Icon, Mark } from "@/components/icons";
import { cx } from "@/components/ui/kit";
import { useCouple } from "@/features/couple/hooks";
import { longDate } from "@/lib/dates";
import { NAV } from "./nav-items";
import { Ribbon } from "./ribbon";
import { routes } from "@/lib/routes";

/** Tablet/desktop nav: the phone tab stood on end as a rail from `md`, the
 *  book's contents page from `lg`. TabBar and this are mutually
 *  `display: none`, which also keeps the hidden one out of the accessibility
 *  tree — never two "Primary" nav landmarks. */
export function SideNav() {
  const path = usePathname();
  const couple = useCouple();
  const me = couple.data?.me.display_name;
  const partner = couple.data?.partner?.display_name;
  const since = couple.data?.started_on;
  return (
    <nav
      aria-label="Primary"
      className="grain sticky top-0 hidden h-[100dvh] shrink-0 flex-col gap-1 border-r border-line bg-bg md:flex md:w-[88px] lg:w-[236px] lg:gap-0"
      style={{ paddingTop: "calc(var(--safe-top) + 20px)", paddingBottom: "calc(var(--safe-bottom) + 24px)" }}
    >
      <Link
        href={routes.home}
        aria-label="Amorae, home"
        className="press flex h-11 items-center gap-2.5 text-ink no-underline md:mb-4 md:justify-center lg:mb-0 lg:justify-start lg:px-6"
      >
        <Mark size={26} className="text-plum" />
        <span className="hidden text-[19px] font-semibold tracking-[-0.02em] lg:inline">
          Amorae
        </span>
      </Link>
      {/* The running head: whose book this is. */}
      {me ? (
        <div className="hidden px-6 pb-2 pt-5 text-micro uppercase text-stone lg:block">
          {partner ? `${me} & ${partner}` : me}
        </div>
      ) : null}
      {NAV.map((item) => {
        const on = item.match(path);
        return (
          <Link
            key={item.href}
            href={item.href}
            aria-current={on ? "page" : undefined}
            className={cx(
              "press relative flex items-center no-underline",
              // Rail: icon over label, centred. Wide: one row, label beside icon.
              "md:h-16 md:flex-col md:justify-center md:gap-1 md:text-micro md:tracking-[0.01em]",
              "lg:h-12 lg:flex-row lg:justify-start lg:gap-3 lg:px-6 lg:text-[16px] lg:tracking-normal",
              on
                ? "bg-paper font-[750] text-ink"
                : "font-semibold text-stone hover:bg-paper hover:text-ink",
            )}
          >
            {on ? <Ribbon dir="in" /> : null}
            <Icon name={item.icon} size={22} strokeWidth={on ? 1.9 : 1.5} />
            <span>{item.label}</span>
          </Link>
        );
      })}
      {since ? (
        <p className="mx-6 mb-0 mt-auto hidden border-t border-line pt-4 font-text text-[15px] leading-[1.4] italic text-stone lg:block">
          Kept together since
          <br />
          {longDate(since)} {since.slice(0, 4)}
        </p>
      ) : null}
    </nav>
  );
}
