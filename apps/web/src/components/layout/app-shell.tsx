"use client";

import { useQueryClient } from "@tanstack/react-query";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { NetworkBanner } from "@/components/layout/network-banner";
import { Main, Screen } from "@/components/layout/screen";
import { SplashView } from "@/components/layout/splash-view";
import { TabBar } from "@/components/layout/tab-bar";
import { ErrorState } from "@/components/ui/kit";
import { useCouple } from "@/features/couple/hooks";
import { isApiError } from "@/lib/api/errors";
import { SESSION_EXPIRED_EVENT } from "@/lib/api/http";
import { clearSignedInState, persistOfflineChanges } from "@/lib/query/persist";
import { routes } from "@/lib/routes";

function unpaired(error: unknown) {
  return isApiError(error) && error.code === "couple_not_found";
}

// The signed-in shell. src/proxy.ts already bounced visitors with no cookie;
// this is the real check: GET /couples/me. 401 → login. couple_not_found →
// pairing. Any other failure (down API, missing route, 500) is a retry
// screen — never a raw "404 page not found".
export function AppShell({ children }: { children: ReactNode }) {
  const couple = useCouple();
  const router = useRouter();
  const path = usePathname();
  const qc = useQueryClient();
  const focused = path.startsWith(routes.prayerMode());

  // Reconnecting needs no wiring here: React Query refetches stale queries
  // and resumes paused changes on its own (lib/query/client.ts). What this
  // does add is keeping those changes across restarts, per user, once we
  // know who's signed in (lib/query/persist.ts).
  const meId = couple.data?.me.id;
  useEffect(() => (meId ? persistOfflineChanges(qc, meId) : undefined), [qc, meId]);

  useEffect(() => {
    const expired = () => { clearSignedInState(qc); router.replace(routes.login({ expired: true })); };
    window.addEventListener(SESSION_EXPIRED_EVENT, expired);
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, expired);
  }, [qc, router]);
  useEffect(() => {
    if (couple.data && !couple.data.onboarding.couple) router.replace(routes.couple());
    if (couple.isError && unpaired(couple.error)) router.replace(routes.couple());
  }, [couple.data, couple.error, couple.isError, router]);

  if (couple.isPending) return <SplashView />;
  if (couple.isError && !unpaired(couple.error)) {
    const network = isApiError(couple.error) && (couple.error.status === 0 || couple.error.code === "network_error");
    return (
      <Screen>
        <Main>
          <ErrorState
            title={network ? "We couldn’t reach the server" : "We couldn’t load your space"}
            text="Nothing of yours is lost. Try again in a moment."
            onRetry={() => { void couple.refetch(); }}
          />
        </Main>
      </Screen>
    );
  }
  if (!couple.data || !couple.data.onboarding.couple) return <SplashView />;
  if (focused) return <>{children}</>;
  return (
    <Screen>
      <div className="shrink-0" style={{ height: "calc(var(--safe-top) + 8px)" }} />
      <NetworkBanner />
      {children}
      <TabBar />
    </Screen>
  );
}
