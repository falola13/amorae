import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { Card } from "@/components/ui/card";
import { isApiError } from "@/lib/api/errors";
import { ProfileForm } from "@/features/users/components/profile-form";
import { getCurrentUser } from "@/features/users/api";

export const metadata: Metadata = { title: "Dashboard" };

export default async function DashboardPage() {
  let user;
  try {
    user = await getCurrentUser();
  } catch (error) {
    // Proxy only checks that a session cookie exists, not that it's still
    // valid — the API is the real authority. A 401 here means the token is
    // expired or revoked, so send the user back through login.
    if (isApiError(error) && error.status === 401) {
      redirect("/login?reason=expired");
    }
    throw error;
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="font-display text-3xl text-fg">Welcome, {user.display_name}</h1>
        <p className="mt-1 text-fg-muted">{user.email}</p>
      </div>
      <Card>
        <h2 className="font-display text-lg text-fg">Your profile</h2>
        <div className="mt-4">
          <ProfileForm displayName={user.display_name} />
        </div>
      </Card>
    </div>
  );
}
