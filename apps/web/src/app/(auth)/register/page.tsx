import type { Metadata } from "next";

import { SafeTop } from "@/components/layout/screen";
import { Para, Title, TopBar } from "@/components/ui/kit";
import { RegisterForm } from "@/features/auth/components/register-form";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Create your account" };

export default function RegisterPage() {
  return (
    <>
      <SafeTop />
      <TopBar back="Back" backHref={routes.welcome} />
      <div className="flex flex-col gap-2 px-6 pb-7 pt-3">
        <Title>Create your account</Title>
        <Para>Your partner will make their own, then you&rsquo;ll link the two.</Para>
      </div>
      <RegisterForm />
    </>
  );
}
