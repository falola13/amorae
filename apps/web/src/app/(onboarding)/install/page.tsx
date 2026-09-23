"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { Icon, type IconName } from "@/components/icons";
import { SafeTop } from "@/components/layout/screen";
import { Button, Para, Title } from "@/components/ui/kit";
import { Bottom } from "@/components/ui/onboarding-bits";
import { useOnboarding } from "@/features/couple/hooks";
import { routes } from "@/lib/routes";
import { useInstall } from "@/lib/pwa/install";
import { useUI } from "@/lib/store/ui";

function Step({
  n,
  icon,
  title,
  text,
  last,
}: {
  n: string;
  icon: IconName;
  title: string;
  text: string;
  last?: boolean;
}) {
  return (
    <li className={`flex min-h-[76px] items-center gap-4 ${last ? "" : "border-b border-line"}`}>
      <span className="tabular w-[22px] text-[13px] font-semibold text-stone">{n}</span>
      <span className="flex grow flex-col gap-0.5">
        <span className="text-bodylg font-semibold">{title}</span>
        <span className="text-support text-stone">{text}</span>
      </span>
      <span className="flex h-11 w-11 items-center justify-center rounded-input border border-line bg-surface">
        <Icon name={icon} size={22} />
      </span>
    </li>
  );
}

// Gentle install guidance (spec 19.2): explain, show the iOS steps or the
// Android prompt, and remember a dismissal so it is not shown again.
export default function InstallPage() {
  const router = useRouter();
  const onboarding = useOnboarding();
  const dismissInstall = useUI((s) => s.dismissInstall);
  const { standalone, ios, prompt } = useInstall();
  const done = () =>
    onboarding.mutate(
      { install: true },
      { onSuccess: () => router.push(routes.notificationsSetup) },
    );
  const later = () => {
    dismissInstall();
    onboarding.mutate(
      { install: true, notifications: true },
      { onSuccess: () => router.replace(routes.home) },
    );
  };

  // Already installed: skip this step. A mutation belongs in an effect, not
  // in the render body — calling it while rendering fires twice under
  // StrictMode and can race with the render that triggered it.
  useEffect(() => {
    if (standalone) done();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [standalone]);

  if (standalone) return null;
  return (
    <>
      <SafeTop />
      <div className="h-11 shrink-0" />
      <div className="flex grow flex-col gap-6 px-6 pt-3">
        <div className="flex flex-col gap-2">
          <Title>Keep Amorae close.</Title>
          <Para>
            Add Amorae to your Home Screen for a more private, app-like experience. It also works
            offline, and it&rsquo;s how your iPhone allows reminders.
          </Para>
        </div>
        {ios || !prompt ? (
          <ol className="m-0 list-none border-y border-line p-0">
            <Step n="01" icon="share" title="Tap Share" text="In Safari’s toolbar, below." />
            <Step
              n="02"
              icon="addsq"
              title="Add to Home Screen"
              text="Scroll the list to find it."
            />
            <Step
              n="03"
              icon="check"
              title="Tap Add"
              text="Then open Amorae from your Home Screen."
              last
            />
          </ol>
        ) : (
          <Button
            icon="addsq"
            onClick={async () => {
              await prompt();
            }}
          >
            Install Amorae
          </Button>
        )}
      </div>
      <Bottom>
        <Button onClick={done} loading={onboarding.isPending}>
          I&rsquo;ve added it
        </Button>
        <Button variant="text" onClick={later}>
          Maybe later
        </Button>
        {ios ? (
          <div className="flex items-center justify-center gap-1.5 pt-1 text-[13px] text-stone">
            <Icon name="down" size={16} />
            Share is in the bar below
          </div>
        ) : null}
      </Bottom>
    </>
  );
}
