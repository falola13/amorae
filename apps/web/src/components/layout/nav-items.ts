import type { IconName } from "@/components/icons";
import { routes } from "@/lib/routes";

/**
 * The primary destinations, in one place: the bottom tab bar uses them on a
 * phone, the side nav on a tablet or desktop. Same order, same labels.
 */
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
