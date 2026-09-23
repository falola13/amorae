"use client";

import { useCouple } from "@/features/couple/hooks";
import { useWeek } from "@/features/prayers/hooks";
import { Main } from "@/components/layout/screen";
import { LinkButton, Para, Segments, Title } from "@/components/ui/kit";
import { routes } from "@/lib/routes";

export default function Completion() {
  const week = useWeek();
  const couple = useCouple();
  const n = week.data?.points.length ?? 5;
  const partner = couple.data?.partner?.display_name ?? "Your partner";
  return (
    <>
      <div className="h-11 shrink-0" />
      <Main className="justify-center gap-5 px-8 pb-6">
        <svg
          width="56"
          height="56"
          viewBox="0 0 56 56"
          aria-hidden="true"
          fill="none"
          stroke="#3F6B52"
          strokeWidth="1.6"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <circle cx="28" cy="28" r="26" opacity="0.35" />
          <path
            d="M18 28.5l7 7 13.5-14"
            strokeWidth="2"
            className="draw-check"
            style={{ animation: "draw 0.5s 0.15s ease-out forwards" }}
          />
        </svg>
        <Title size="lg">You&rsquo;ve prayed through this week.</Title>
        <Para size="lg">
          All {n} prayers, in your own time. {partner} will see that you&rsquo;ve finished.
        </Para>
        <Segments total={n} done={n} color="bg-green" height={4} className="mt-1 w-32" />
      </Main>
      <div className="flex shrink-0 flex-col gap-2 px-6 pt-4 pb-3">
        <LinkButton href={routes.home} replace>
          Done
        </LinkButton>
        <LinkButton href={routes.history} variant="text" icon="pencil">
          Add a reflection
        </LinkButton>
      </div>
    </>
  );
}
