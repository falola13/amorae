import type { Metadata } from "next";

import { Alert } from "@/components/ui/alert";
import { Card } from "@/components/ui/card";
import { LoginForm } from "@/features/auth/components/login-form";
import { safeNext } from "@/lib/safe-next";

export const metadata: Metadata = { title: "Sign in" };

interface LoginPageProps {
  searchParams: Promise<{ next?: string; reason?: string }>;
}

export default async function LoginPage({ searchParams }: LoginPageProps) {
  const params = await searchParams;
  const next = safeNext(params.next);
  const expired = params.reason === "expired";

  return (
    <Card>
      <h1 className="font-display text-2xl text-fg">Welcome back</h1>
      <p className="mt-1 text-sm text-fg-muted">Sign in to your Amorae account.</p>
      {expired ? (
        <div className="mt-4">
          <Alert message="Your session expired. Please sign in again." />
        </div>
      ) : null}
      <div className="mt-6">
        <LoginForm next={next} />
      </div>
    </Card>
  );
}
