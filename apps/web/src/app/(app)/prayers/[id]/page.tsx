"use client";

import { useParams } from "next/navigation";

import { Main } from "@/components/layout/screen";
import { BottomActions, Button, LinkButton, Skeleton, Title, TopBar } from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { Scripture } from "@/components/ui/scripture";
import { useCouple } from "@/features/couple/hooks";
import { useSetCompleted, useWeek } from "@/features/prayers/hooks";
import { routes } from "@/lib/routes";

export default function PrayerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const week = useWeek(); const couple = useCouple(); const complete = useSetCompleted();
  const loading = <Main><div className="pt-4"><Skeleton /></div></Main>;

  return (
    <QueryState queries={[week]} loading={loading}>
      {(w) => {
        const p = w.points.find((x) => x.id === id);
        if (!p) return loading;
        const idx = w.points.findIndex((x) => x.id === id);
        const done = w.my_completed.includes(p.id);
        const setter = w.setter_id === couple.data?.me.id ? "you" : (couple.data?.partner?.display_name ?? "your partner");
        return (
          <>
            <TopBar back="Prayers" backHref={routes.prayers} right={<div className="tabular pr-3 text-support font-semibold text-stone">{idx + 1} of {w.points.length}</div>} />
            <Main className="gap-5 pt-5">
              <div className="flex flex-col gap-2.5"><Title size="lg">{p.title}</Title><div className="text-support text-stone">Added by {setter} on Sunday</div></div>
              {p.text ? <p className="m-0 text-[19px] leading-[1.6]">{p.text}</p> : null}
              {p.scripture ? <Scripture reference={p.scripture} verse={p.verse} /> : null}
            </Main>
            <BottomActions>
              {done ? <Button variant="secondary" icon="check" onClick={() => complete.mutate({ pointId: p.id, done: false })}>Prayed &middot; tap to undo</Button>
                : <Button icon="check" onClick={() => complete.mutate({ pointId: p.id, done: true })}>I&rsquo;ve prayed</Button>}
              <LinkButton href={routes.prayerMode({ at: idx })} variant="text" icon="moon">Open in prayer mode</LinkButton>
            </BottomActions>
          </>
        );
      }}
    </QueryState>
  );
}
