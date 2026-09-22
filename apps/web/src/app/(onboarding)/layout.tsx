import type { ReactNode } from "react";

import { Screen } from "@/components/layout/screen";

export default function OnboardingLayout({ children }: { children: ReactNode }) {
  return <Screen>{children}</Screen>;
}
