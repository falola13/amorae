import type { ReactNode } from "react";

import { Screen } from "@/components/layout/screen";

// Terms and Privacy: public, readable signed out (src/proxy.ts lists them).
export default function LegalLayout({ children }: { children: ReactNode }) {
  return <Screen>{children}</Screen>;
}
