import type { IconName } from "@/components/icons";
import { routes } from "@/lib/routes";

/** Shared by TabBar (phone) and SideNav (tablet/desktop). */
export const NAV: { label: string; icon: IconName; href: string; match: (p: string) => boolean }[] =
  [
    { label: "Home", icon: "home", href: routes.home, match: (p) => p === "/" },
    {
      label: "Together",
      icon: "together",
      href: routes.together,
      match: (p) => p.startsWith("/together"),
    },
    {
      label: "Prayers",
      icon: "book",
      href: routes.prayers,
      match: (p) => p.startsWith("/prayers"),
    },
    {
      label: "History",
      icon: "clock",
      href: routes.history,
      match: (p) => p.startsWith("/history"),
    },
    {
      label: "Settings",
      icon: "sliders",
      href: routes.settings,
      match: (p) => p.startsWith("/settings"),
    },
  ];
