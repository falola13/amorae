"use client";

import { useParams } from "next/navigation";

import { useCouple } from "@/features/couple/hooks";
import { ChallengeActive } from "@/features/together/components/challenge-active";
import { ChallengeFinish } from "@/features/together/components/challenge-finish";
import { isOver } from "@/features/together/challenges";
import { useChallenge, useChallengeById } from "@/features/together/hooks";
import { isApiError } from "@/lib/api/errors";
import { routes } from "@/lib/routes";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import { Skeleton, TopBar } from "@/components/ui/kit";

/** One of the couple's challenges, past or present. Over ones are read-only but for your reflection. */
export default function ChallengeDetail() {
  const { id } = useParams<{ id: string }>();
  const ch = useChallengeById(id);
  const current = useChallenge();
  const couple = useCouple();
  const partner = couple.data?.partner?.display_name ?? "They";
  // Starting another only works when nothing is running.
  const nothingRunning = isApiError(current.error) && current.error.code === "challenge_not_found";

  return (
    <>
      <TopBar back="Challenges" backHref={routes.challenges} />
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
            <ChallengeFinish c={c} partner={partner} canStartAnother={nothingRunning} />
          ) : (
            <ChallengeActive c={c} partner={partner} />
          )
        }
      </QueryState>
    </>
  );
}
