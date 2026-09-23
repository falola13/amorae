"use client";

import { useState } from "react";
import { useMilestones } from "@/features/together/hooks";
import { MilestoneComposer } from "@/features/together/components/milestone-composer";
import { nextOccurrence } from "@/features/together/milestones";
import { daysUntil, iso } from "@/lib/dates";
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
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";

export default function Milestones() {
  const ms = useMilestones();
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
                        right={`in ${daysUntil(d.next, today())} days`}
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
                      />
                    ))}
                  </Section>
                ) : null}
              </>
            );
          }}
        </QueryState>
      </Main>
      <BottomActions>
        <Button icon="plus" onClick={() => setOpen(true)}>
          Add a date
        </Button>
      </BottomActions>
      <MilestoneComposer open={open} onClose={() => setOpen(false)} />
    </>
  );
}
