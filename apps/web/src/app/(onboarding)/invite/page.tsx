"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { SafeTop } from "@/components/layout/screen";
import { Button, LinkButton, Micro, Para, Title, TopBar } from "@/components/ui/kit";
import { Bottom } from "@/components/ui/onboarding-bits";
import { useCouple, useRegenerateInvite } from "@/features/couple/hooks";
import { formatInviteCode } from "@/lib/invite-code";
import { routes } from "@/lib/routes";

/** While waiting, ask the API again this often, so the screen notices the join. */
const POLL_MS = 10_000;

export default function InvitePartnerPage() {
  const couple = useCouple();
  const regenerate = useRegenerateInvite();
  const router = useRouter();
  const code = couple.data?.invite_code ? formatInviteCode(couple.data.invite_code) : "•••-•••";
  const [copied, setCopied] = useState(false);

  const partner = couple.data?.partner?.display_name;
  const refetch = couple.refetch;
  useEffect(() => {
    if (partner) return;
    const timer = setInterval(() => void refetch(), POLL_MS);
    return () => clearInterval(timer);
  }, [partner, refetch]);
  useEffect(() => {
    if (partner) router.replace(routes.home);
  }, [partner, router]);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard blocked */
    }
  };
  const share = async () => {
    const data = {
      title: "Amorae",
      text: `Join me on Amorae, our private space. Invitation code: ${code}`,
      url: `${location.origin}${routes.join({ code })}`,
    };
    if (navigator.share) {
      try {
        await navigator.share(data);
      } catch {
        /* cancelled */
      }
    } else copy();
  };
  return (
    <>
      <SafeTop />
      {/* Back leads to the app, not earlier signup steps; invite is reachable again via Settings → Couple. */}
      <TopBar back="Our space" backHref={routes.home} />
      <div className="flex grow flex-col gap-6 px-6 pt-3">
        <div className="flex flex-col gap-2">
          <Micro>Step 3 of 3</Micro>
          <Title>Invite your partner</Title>
          <Para>Send this to one person. It works once, and expires in 7 days.</Para>
        </div>
        <div className="flex flex-col items-center gap-1.5 rounded-card border border-line bg-surface px-4 pb-3 pt-7">
          <div className="text-[13px] font-semibold text-stone">Your invitation code</div>
          <div className="tabular text-[30px] font-semibold tracking-[0.14em]" data-selectable>
            {code}
          </div>
          <Button
            variant="text"
            icon={copied ? "check" : "copy"}
            onClick={copy}
            className="w-auto px-4"
          >
            {copied ? "Copied" : "Copy code"}
          </Button>
          {/* Regenerating invalidates the old code immediately. */}
          <Button
            variant="text"
            onClick={() => regenerate.mutate()}
            loading={regenerate.isPending}
            className="w-auto px-4"
          >
            New code
          </Button>
        </div>
        <div role="status" className="flex items-center gap-2.5 text-support text-stone">
          <span className="h-2 w-2 animate-breathe rounded-full bg-amber" />
          Waiting for your partner to join. This screen moves on by itself when they do.
        </div>
      </div>
      <Bottom>
        <Button icon="share" onClick={share}>
          Share invitation
        </Button>
        <LinkButton href={routes.install} variant="text">
          Continue for now
        </LinkButton>
        <LinkButton href={routes.home} variant="text">
          Go to Amorae
        </LinkButton>
      </Bottom>
    </>
  );
}
