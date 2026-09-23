import type { Metadata } from "next";

import { Icon, Mark } from "@/components/icons";
import { SafeTop } from "@/components/layout/screen";
import { Bottom } from "@/components/ui/onboarding-bits";
import { LinkButton, Micro, Para, Title } from "@/components/ui/kit";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Welcome" };

export default function WelcomePage() {
  return (
    <>
      <SafeTop />
      <div className="flex h-11 shrink-0 items-center gap-2 px-6">
        <Mark size={28} className="text-plum" />
        <span className="text-[18px] font-semibold tracking-[-0.02em]">Amorae</span>
      </div>
      <div className="flex grow flex-col justify-end gap-4 px-6 pb-7">
        <Micro>Two hearts, one faith</Micro>
        <Title size="display">A private space for the life you&rsquo;re building together.</Title>
        <Para size="lg">
          Plan, pray, remember and grow. One quiet place that belongs to the two of you.
        </Para>
      </div>
      <Bottom>
        <LinkButton href={routes.register}>Create an account</LinkButton>
        <LinkButton href={routes.login()} variant="secondary">
          I already have an account
        </LinkButton>
        <div className="flex items-center justify-center gap-1.5 pt-2.5 text-[13px] text-stone">
          <Icon name="lock" size={16} />
          Just the two of you. Nothing here is public.
        </div>
      </Bottom>
    </>
  );
}
