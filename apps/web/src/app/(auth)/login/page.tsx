import type { Metadata } from "next";

import { Alert } from "@/components/ui/alert";
import { Card } from "@/components/ui/card";
import { LoginForm } from "@/features/auth/components/login-form";

export const metadata: Metadata = { title: "Sign in" };

// Only a same-origin, single-slash path is safe to send the browser to
// after login — mirrors the check in features/auth/actions.ts, which is the
// one that actually enforces it (this copy only decides what's shown in the
// form's hidden `next` field).
function safeNext(value: string | undefined): string {
  if (!value) return "/dashboard";
  if (!value.startsWith("/") || value.startsWith("//") || value.startsWith("/\\")) {
    return "/dashboard";
  }
  return value;
}

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
