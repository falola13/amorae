"use client";

import { useState } from "react";

import { useCouple } from "@/features/couple/hooks";
import { ChallengeLibrary } from "@/features/together/components/challenge-library";
import { ChallengeRunning } from "@/features/together/components/challenge-running";
import { useActiveChallenges } from "@/features/together/hooks";
import { routes } from "@/lib/routes";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import { Skeleton, TopBar } from "@/components/ui/kit";

export default function Challenges() {
  const active = useActiveChallenges();
  const couple = useCouple();
  const [browsing, setBrowsing] = useState(false);
  const partner = couple.data?.partner?.display_name ?? "They";

  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <QueryState
        queries={[active]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(list) =>
          list.length === 0 ? (
            <ChallengeLibrary />
          ) : browsing ? (
            <ChallengeLibrary onBack={() => setBrowsing(false)} />
          ) : (
            <ChallengeRunning
              list={list}
              partner={partner}
              onStartAnother={() => setBrowsing(true)}
            />
          )
        }
      </QueryState>
    </>
  );
}
