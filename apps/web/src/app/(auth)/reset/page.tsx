import type { Metadata } from "next";

import { SafeTop } from "@/components/layout/screen";
import { Para, Title, TopBar } from "@/components/ui/kit";
import { ResetForm } from "@/features/auth/components/reset-form";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Choose a new password" };

export default async function ResetPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>;
}) {
  const { token } = await searchParams;
  return (
    <>
      <SafeTop />
      <TopBar back="Log in" backHref={routes.login()} />
      <div className="flex flex-col gap-2 px-6 pb-7 pt-3">
        <Title>Choose a new password</Title>
        <Para>You&rsquo;ll be signed out everywhere, then you can log in with it.</Para>
      </div>
      <ResetForm token={token ?? ""} />
    </>
  );
}
