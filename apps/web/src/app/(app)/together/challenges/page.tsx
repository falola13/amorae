"use client";

import { useCouple } from "@/features/couple/hooks";
import { ChallengeActive } from "@/features/together/components/challenge-active";
import { ChallengeFinish } from "@/features/together/components/challenge-finish";
import { ChallengeLibrary } from "@/features/together/components/challenge-library";
import { isOver } from "@/features/together/challenges";
import { useChallenge } from "@/features/together/hooks";
import { isApiError } from "@/lib/api/errors";
import { routes } from "@/lib/routes";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import { Skeleton, TopBar } from "@/components/ui/kit";

export default function Challenges() {
  const ch = useChallenge();
  const couple = useCouple();
  const partner = couple.data?.partner?.display_name ?? "They";

  // No active challenge is a 404, not an error state: offer to start one.
  if (isApiError(ch.error) && ch.error.code === "challenge_not_found") {
    return (
      <>
        <TopBar back="Our space" backHref={routes.together} />
        <ChallengeLibrary />
      </>
    );
  }
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <QueryState
        queries={[ch]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(c) =>
          isOver(c.status) ? (
            <ChallengeFinish c={c} partner={partner} canStartAnother />
          ) : (
            <ChallengeActive c={c} partner={partner} />
          )
        }
      </QueryState>
    </>
  );
}
