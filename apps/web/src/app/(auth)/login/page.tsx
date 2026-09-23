import type { Metadata } from "next";

import { SafeTop } from "@/components/layout/screen";
import { Para, Title, TopBar } from "@/components/ui/kit";
import { LoginForm } from "@/features/auth/components/login-form";
import { safeNext } from "@/lib/safe-next";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Log in" };

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ next?: string; reason?: string }>;
}) {
  const params = await searchParams;
  return (
    <>
      <SafeTop />
      <TopBar back="Back" backHref={routes.welcome} />
      <div className="flex flex-col gap-2 px-6 pb-7 pt-3">
        <Title>Welcome back</Title>
        <Para>Everything is where the two of you left it.</Para>
      </div>
      <LoginForm next={safeNext(params.next)} expired={params.reason === "expired"} />
    </>
  );
}
