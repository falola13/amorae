"use client";

import { useState } from "react";
import { useDeleteMilestone, useMilestones } from "@/features/together/hooks";
import { MilestoneComposer } from "@/features/together/components/milestone-composer";
import { countdown, nextOccurrence } from "@/features/together/milestones";
import { iso } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { DateRow } from "@/components/ui/date-row";
import { today } from "@/lib/today";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import {
  BottomActions,
  Button,
  EmptyState,
  Para,
  Section,
  Sheet,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import type { Milestone } from "@/lib/api/types";

function MilestoneActions({ date, onClose }: { date: Milestone | null; onClose: () => void }) {
  const remove = useDeleteMilestone();
  if (!date) return null;
  return (
    <Sheet open onClose={onClose} title="Remove this date?" labelledBy="del-date-h">
      <Para>
        “{date.title}” goes for both of you, and any reminder with it. You can always add it again.
      </Para>
      <div className="flex flex-col gap-1">
        <Button
          variant="secondary"
          className="border-red text-red"
          loading={remove.isPending}
          onClick={() => remove.mutate(date.id, { onSuccess: onClose })}
        >
          Remove it
        </Button>
        <Button variant="text" onClick={onClose}>
          Keep it
        </Button>
      </div>
    </Sheet>
  );
}

export default function Milestones() {
  const ms = useMilestones();
  const [acting, setActing] = useState<Milestone | null>(null);
  // The empty state already offers this, centred, with a line saying what
  // it is for. Showing the bar as well put two buttons for the same thing
  // on one screen, one under the other. The bar is for when there is a
  // list to add to.
  const hasAny = (ms.data?.length ?? 0) > 0;
  const [open, setOpen] = useState(false);
  const todayIso = iso(today());
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="pt-2">
          <Title>Milestones</Title>
        </div>
        <QueryState queries={[ms]} loading={<Skeleton />}>
          {(milestones) => {
            if (milestones.length === 0) {
              return (
                <EmptyState
                  ghost="dots"
                  title="No dates yet."
                  text="Anniversaries, first dates, the ones worth celebrating again next year."
                  cta={
                    <Button icon="plus" onClick={() => setOpen(true)}>
                      Add a date
                    </Button>
                  }
                />
              );
            }
            const upcoming = milestones
              .filter((d) => d.reminder)
              .map((d) => ({ ...d, next: nextOccurrence(d.date, todayIso) }))
              .sort((a, b) => a.next.localeCompare(b.next));
            const story = milestones
              .filter((d) => !d.reminder)
              .sort((a, b) => b.date.localeCompare(a.date));
            return (
              <>
                <Para className="mt-1.5">The dates that matter to us.</Para>
                {upcoming.length ? (
                  <Section label="Coming up" className="mt-[18px]">
                    {upcoming.map((d) => (
                      <DateRow
                        key={d.id}
                        date={d.next}
                        title={d.title}
                        sub={d.sub}
                        right={countdown(d.next, today())}
                        onAction={() => setActing(d)}
                      />
                    ))}
                  </Section>
                ) : null}
                {story.length ? (
                  <Section label="Our story so far" className="mb-4 mt-6">
                    {story.map((d) => (
                      <DateRow
                        key={d.id}
                        date={d.date}
                        title={d.title}
                        sub={d.sub ?? d.date.slice(0, 4)}
                        onAction={() => setActing(d)}
                      />
                    ))}
                  </Section>
                ) : null}
              </>
            );
          }}
        </QueryState>
      </Main>
      {hasAny ? (
        <BottomActions>
          <Button icon="plus" onClick={() => setOpen(true)}>
            Add a date
          </Button>
        </BottomActions>
      ) : null}
      <MilestoneComposer open={open} onClose={() => setOpen(false)} />
      <MilestoneActions date={acting} onClose={() => setActing(null)} />
    </>
  );
}
