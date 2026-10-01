"use client";

import { useParams } from "next/navigation";

import { useCouple } from "@/features/couple/hooks";
import { ChallengeActive } from "@/features/together/components/challenge-active";
import { ChallengeFinish } from "@/features/together/components/challenge-finish";
import { MAX_ACTIVE, isOver } from "@/features/together/challenges";
import { useActiveChallenges, useChallengeById } from "@/features/together/hooks";
import { routes } from "@/lib/routes";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import { Skeleton, TopBar } from "@/components/ui/kit";

/** One of the couple's challenges, running, finished or ended. Over ones are read-only but for your reflection. */
export default function ChallengeDetail() {
  const { id } = useParams<{ id: string }>();
  const ch = useChallengeById(id);
  const active = useActiveChallenges();
  const couple = useCouple();
  const partner = couple.data?.partner?.display_name ?? "They";
  // Starting another needs a free place among the running ones.
  const roomToStart = active.data !== undefined && active.data.length < MAX_ACTIVE;

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
            <ChallengeFinish c={c} partner={partner} canStartAnother={roomToStart} />
          ) : (
            <ChallengeActive c={c} partner={partner} />
          )
        }
      </QueryState>
    </>
  );
}
