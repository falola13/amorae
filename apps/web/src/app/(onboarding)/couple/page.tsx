"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { Icon } from "@/components/icons";
import { SafeTop } from "@/components/layout/screen";
import { Button, LinkButton, Micro, Para, Title, TopBar } from "@/components/ui/kit";
import { Bottom, Fact } from "@/components/ui/onboarding-bits";
import { logoutAction } from "@/features/auth/actions";
import { useCouple, useCreateCouple, useEndedCouples, useMe } from "@/features/couple/hooks";
import { clearSignedInState } from "@/lib/query/persist";
import { routes } from "@/lib/routes";
import { zoneLabel } from "@/lib/timezones";

export default function CreateCouplePage() {
  const router = useRouter();
  const create = useCreateCouple();
  const couple = useCouple();
  const me = useMe();
  // Surfaces the past space link here, since ending a space also lands you on this screen.
  const ended = useEndedCouples();
  const pastSpace = ended.data?.[0];
  const qc = useQueryClient();

  // Redirects away if a couple already exists (back button, bookmark, second tab), since the API refuses to create a second one.
  const paired = Boolean(couple.data?.partner);
  const hasCouple = Boolean(couple.data);
  useEffect(() => {
    if (paired) router.replace(routes.home);
    else if (hasCouple) router.replace(routes.invite);
  }, [paired, hasCouple, router]);

  return (
    <>
      <SafeTop />
      {/* No back navigation: this is the first step after signup, so logging out is the only way out. */}
      <TopBar />
      <div className="flex grow flex-col gap-5 px-6 pt-3">
        <div className="flex flex-col gap-2">
          <Micro>Step 2 of 3</Micro>
          <Title>Start a space for the two of you</Title>
          <Para>You&rsquo;ll get one private invitation to send your partner.</Para>
        </div>
        <div className="flex flex-col border-t border-line">
          <Fact
            icon="users"
            title="Always exactly two"
            text="Only you and your partner can see what’s written here."
          />
          <Fact
            icon="clock"
            title="A prayer week every Sunday"
            text="You’ll set the first week. After that it alternates."
          />
          {/* Seeds the couple's timezone from the creator's own; changeable later in the space itself (DEC-27). */}
          <Fact
            icon="globe"
            title={zoneLabel(me.data?.timezone ?? "UTC")}
            text="Your prayer week turns over on Sunday here. You can change it later."
          />
        </div>
      </div>
      <Bottom>
        <Button
          loading={create.isPending}
          onClick={() => create.mutate(undefined, { onSuccess: () => router.push(routes.invite) })}
        >
          Create our space
        </Button>
        <LinkButton href={routes.join()} variant="text">
          I have an invitation code
        </LinkButton>
        {pastSpace ? (
          <LinkButton href={routes.settingsPastSpace} variant="text">
            Your past space
          </LinkButton>
        ) : null}
        <Button
          variant="text"
          onClick={async () => {
            clearSignedInState(qc);
            await logoutAction();
          }}
        >
          Not you? Log out
        </Button>
      </Bottom>
      <span className="hidden">
        <Icon name="users" />
      </span>
    </>
  );
}
