import { AppShell } from "@/components/layout/app-shell";
import { HomeScreen } from "@/features/home/home-screen";

// Home lives at "/" outside the (app) route group, so it wraps itself in the group's shell.
export default function HomePage() {
  return (
    <AppShell>
      <HomeScreen />
    </AppShell>
  );
}
