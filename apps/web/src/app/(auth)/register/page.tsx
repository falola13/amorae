import type { Metadata } from "next";

import { Card } from "@/components/ui/card";
import { RegisterForm } from "@/features/auth/components/register-form";

export const metadata: Metadata = { title: "Register" };

export default function RegisterPage() {
  return (
    <Card>
      <h1 className="font-display text-2xl text-fg">Create your account</h1>
      <p className="mt-1 text-sm text-fg-muted">It takes less than a minute.</p>
      <div className="mt-6">
        <RegisterForm />
      </div>
    </Card>
  );
}
