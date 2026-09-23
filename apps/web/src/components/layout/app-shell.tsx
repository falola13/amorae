"use client";

import { useQueryClient } from "@tanstack/react-query";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { NetworkBanner } from "@/components/layout/network-banner";
import { Main, Screen } from "@/components/layout/screen";
import { SideNav } from "@/components/layout/side-nav";
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

// Screens that stand on their own account rather than on a couple. Someone
// who has just ended their space still has to be able to reach their devices,
// their data and the way out.
const ACCOUNT_ROUTES: string[] = [
  routes.settings,
  routes.settingsProfile,
  routes.settingsDevices,
  routes.settingsPastSpace,
];

// The signed-in shell. src/proxy.ts already bounced visitors with no cookie;
// this is the real check: GET /couples/me. 401 → login. couple_not_found →
// pairing. Any other failure (down API, missing route, 500) before the couple
// has ever loaded is a retry screen — never a raw "404 page not found".
export function AppShell({ children }: { children: ReactNode }) {
  const couple = useCouple();
  const router = useRouter();
  const path = usePathname();
  const qc = useQueryClient();
  const focused = path.startsWith(routes.prayerMode());
  // Your account is yours before you have a space and after you leave one.
  // These screens are about you — what is left of an ended space, your
  // devices, your data, your profile — so they are not sent back to pairing
  // when there is no couple. Everything else needs one.
  const outlivesCouple = ACCOUNT_ROUTES.includes(path);

  // Reconnecting needs no wiring here: React Query refetches stale queries
  // and resumes paused changes on its own (lib/query/client.ts). What this
  // does add is keeping those changes across restarts, per user, once we
  // know who's signed in (lib/query/persist.ts).
  const meId = couple.data?.me.id;
  useEffect(() => (meId ? persistOfflineChanges(qc, meId) : undefined), [qc, meId]);

  useEffect(() => {
    const expired = () => {
      clearSignedInState(qc);
      router.replace(routes.login({ expired: true }));
    };
    window.addEventListener(SESSION_EXPIRED_EVENT, expired);
    return () => window.removeEventListener(SESSION_EXPIRED_EVENT, expired);
  }, [qc, router]);
  useEffect(() => {
    if (outlivesCouple) return;
    if (couple.data && !couple.data.onboarding.couple) router.replace(routes.couple());
    if (couple.isError && unpaired(couple.error)) router.replace(routes.couple());
  }, [outlivesCouple, couple.data, couple.error, couple.isError, router]);

  if (couple.isPending && !outlivesCouple) return <SplashView />;
  // Whatever went wrong, it wasn't this screen's business: what is left of an
  // ended space doesn't depend on having a live one.
  if (outlivesCouple && !couple.data) return <>{children}</>;
  // Full screen only when there's nothing to show. With the couple already
  // loaded, a failed background refetch keeps the app usable; each screen's
  // own queries show their problems inline.
  if (!couple.data && couple.isError && !unpaired(couple.error)) {
    const network =
      isApiError(couple.error) &&
      (couple.error.status === 0 || couple.error.code === "network_error");
    return (
      <Screen>
        <Main>
          <ErrorState
            title={network ? "We couldn’t reach the server" : "We couldn’t load your space"}
            text="Nothing of yours is lost. Try again in a moment."
            onRetry={() => {
              void couple.refetch();
            }}
          />
        </Main>
      </Screen>
    );
  }
  if (!couple.data || !couple.data.onboarding.couple) return <SplashView />;
  if (focused) return <>{children}</>;
  // One layout, two shapes: a phone gets the column and the bottom tab bar; a
  // tablet or desktop gets the side nav and a wider column, centred in what's
  // left. Only one navigation is ever rendered (see SideNav).
  return (
    <div className="mx-auto flex w-full max-w-[1280px]">
      <SideNav />
      {/* min-w-0 so the column measures the space left by the nav, not the viewport. */}
      <div className="flex min-w-0 flex-1 justify-center">
        <Screen size="app">
          <div className="shrink-0" style={{ height: "calc(var(--safe-top) + 8px)" }} />
          <NetworkBanner />
          {children}
          <TabBar />
        </Screen>
      </div>
    </div>
  );
}
