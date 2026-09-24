"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Icon } from "@/components/icons";
import { SafeTop } from "@/components/layout/screen";
import { Button, Para, Title } from "@/components/ui/kit";
import { Bottom } from "@/components/ui/onboarding-bits";
import { useOnboarding } from "@/features/couple/hooks";
import { routes } from "@/lib/routes";
import { usePush } from "@/lib/pwa/push";

function Preview({ when, text }: { when: string; text: string }) {
  return (
    <div className="flex flex-col gap-0.5 border-b border-line py-3.5">
      <div className="text-[13px] font-semibold text-stone">{when}</div>
      <div className="text-[16px] font-medium">{text}</div>
    </div>
  );
}

// Explain first, then ask (spec 19.3). "Not now" is a first-class choice.
export default function NotificationPermissionPage() {
  const router = useRouter();
  const onboarding = useOnboarding();
  const { request } = usePush();
  const [busy, setBusy] = useState(false);
  const finish = () =>
    onboarding.mutate({ notifications: true }, { onSuccess: () => router.replace(routes.home) });
  const enable = async () => {
    setBusy(true);
    try {
      await request();
    } catch {
      /* permission request failed; still continue onboarding */
    } finally {
      setBusy(false);
    }
    finish();
  };
  return (
    <>
      <SafeTop />
      <div className="h-11 shrink-0" />
      <div className="flex grow flex-col gap-6 px-6 pt-3">
        <Icon name="bell" size={28} strokeWidth={1.4} className="text-plum" />
        <div className="flex flex-col gap-2">
          <Title>Stay close to what you&rsquo;ve planned</Title>
          <Para>A few calm reminders about your shared life, and nothing else. For example:</Para>
        </div>
        <div className="flex flex-col border-t border-line">
          <Preview when="On Sundays" text="Your new prayer week is ready." />
          {/* What actually arrives: the plan's own name and when it is, and
              nothing else about it (FR-NOTF-005.AC2). */}
          <Preview
            when="Before something you’ve planned"
            text="Dinner at Terra — today at 7:00 pm."
          />
          <Preview
            when="When your partner shares something"
            text="Adeola added something to your shared journal."
          />
        </div>
        <Para size="support">You choose which ones you get, anytime, in Settings.</Para>
      </div>
      <Bottom>
        <Button onClick={enable} loading={busy}>
          Enable notifications
        </Button>
        <Button variant="text" onClick={finish}>
          Not now
        </Button>
      </Bottom>
    </>
  );
}
