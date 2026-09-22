"use client";

import { useRouter } from "next/navigation";

import { Icon } from "@/components/icons";
import { SafeTop } from "@/components/layout/screen";
import { Button, LinkButton, Micro, Para, Title, TopBar } from "@/components/ui/kit";
import { Bottom, Fact } from "@/components/ui/onboarding-bits";
import { useCreateCouple } from "@/features/couple/hooks";
import { routes } from "@/lib/routes";

export default function CreateCouplePage() {
  const router = useRouter();
  const create = useCreateCouple();
  return (
    <>
      <SafeTop />
      <TopBar back="Back" backHref={routes.register} />
      <div className="flex grow flex-col gap-5 px-6 pt-3">
        <div className="flex flex-col gap-2"><Micro>Step 2 of 3</Micro><Title>Start a space for the two of you</Title><Para>You&rsquo;ll get one private invitation to send your partner.</Para></div>
        <div className="flex flex-col border-t border-line">
          <Fact icon="users" title="Always exactly two" text="Only you and your partner can see what’s written here." />
          <Fact icon="clock" title="A prayer week every Sunday" text="You’ll set the first week. After that it alternates." />
          <Fact icon="globe" title="Lagos time (WAT)" text="Your weeks follow this timezone." trailing={<button type="button" className="press h-11 px-1 text-[15px] font-semibold text-plum">Change</button>} />
        </div>
      </div>
      <Bottom>
        <Button loading={create.isPending} onClick={() => create.mutate(undefined, { onSuccess: () => router.push(routes.invite) })}>Create our space</Button>
        <LinkButton href={routes.join()} variant="text">I have an invitation code</LinkButton>
      </Bottom>
      <span className="hidden"><Icon name="users" /></span>
    </>
  );
}
