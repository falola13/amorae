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

// Screens tied to the account rather than a couple, reachable even after a couple ends.
const ACCOUNT_ROUTES: string[] = [
  routes.settings,
  routes.settingsProfile,
  routes.settingsDevices,
  // Reminders are per person, not per couple, but Settings links here too.
  routes.settingsNotifications,
  routes.settingsPastSpace,
];

// Real auth check (proxy.ts only checks for a cookie): GET /couples/me. 401 →
// login, couple_not_found → pairing, other failures before load → retry screen.
export function AppShell({ children }: { children: ReactNode }) {
  const couple = useCouple();
  const router = useRouter();
  const path = usePathname();
  const qc = useQueryClient();
  const focused = path.startsWith(routes.prayerMode());
  const outlivesCouple = ACCOUNT_ROUTES.includes(path);

  // React Query handles reconnect/refetch itself (lib/query/client.ts); this
  // only persists paused changes across restarts, per user (lib/query/persist.ts).
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
  // An ended space doesn't need a live couple to render.
  if (outlivesCouple && !couple.data) return <>{children}</>;
  // Full-screen error only when there's nothing to show; once loaded, a failed
  // background refetch keeps the app usable and each screen shows its own errors.
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
  // Phone: column + bottom tab bar. Tablet/desktop: side nav + wider column.
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
