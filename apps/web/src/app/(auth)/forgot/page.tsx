import type { Metadata } from "next";

import { SafeTop } from "@/components/layout/screen";
import { Para, Title, TopBar } from "@/components/ui/kit";
import { ForgotForm } from "@/features/auth/components/forgot-form";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Reset your password" };

export default function ForgotPage() {
  return (
    <>
      <SafeTop />
      <TopBar back="Log in" backHref={routes.login()} />
      <div className="flex flex-col gap-2 px-6 pb-7 pt-3">
        <Title>Forgot your password?</Title>
        <Para>Enter your email and we&rsquo;ll send you a link to choose a new one.</Para>
      </div>
      <ForgotForm />
    </>
  );
}
