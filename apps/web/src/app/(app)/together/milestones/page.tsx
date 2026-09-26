"use client";

import Link from "next/link";
import { useState } from "react";
import { useCouple, useMe } from "@/features/couple/hooks";
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

/** "From your space" / "From your profile" / "From {partner}'s profile" for one the
 *  server derived; null for one someone added by hand, which gets no subline at all. */
function derivedSub(m: Milestone, meId: string | undefined, partner: string): string | null {
  if (m.source === "anniversary") return "From your space";
  if (m.source === "birthday")
    return m.about === meId ? "From your profile" : `From ${partner}’s profile`;
  return null;
}

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
          onClick={() => remove.mutate(date.id, { onSuccess: onClose, onQueued: onClose })}
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
  const couple = useCouple();
  const me = useMe();
  const [acting, setActing] = useState<Milestone | null>(null);
  // Hides the bottom add-button when the empty state already shows one.
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
        <QueryState queries={[ms, couple, me]} loading={<Skeleton />}>
          {(milestones, coupleData, meData) => {
            const meId = coupleData.me.id;
            const partner = coupleData.partner?.display_name ?? "your partner";
            const noAnniversary = !coupleData.started_on;
            const noBirthday = !meData.birthday;

            const prompts =
              noAnniversary || noBirthday ? (
                <div className="flex flex-col gap-1 pt-1.5">
                  {noAnniversary ? (
                    <Link
                      href={routes.settingsCouple}
                      className="press text-support text-plum no-underline"
                    >
                      Add when you got together and your anniversary appears here.
                    </Link>
                  ) : null}
                  {noBirthday ? (
                    <Link
                      href={routes.settingsProfile}
                      className="press text-support text-plum no-underline"
                    >
                      Add your birthday so {partner} never misses it.
                    </Link>
                  ) : null}
                </div>
              ) : null;

            if (milestones.length === 0) {
              return (
                <>
                  {prompts}
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
                </>
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
                {prompts}
                {upcoming.length ? (
                  <Section label="Coming up" className="mt-[18px]">
                    {upcoming.map((d) => (
                      <DateRow
                        key={d.id}
                        date={d.next}
                        title={d.title}
                        sub={derivedSub(d, meId, partner) ?? d.sub}
                        right={countdown(d.next, today())}
                        onAction={d.source ? undefined : () => setActing(d)}
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
                        sub={
                          derivedSub(d, meId, partner) ??
                          d.sub ??
                          (d.year_known === false ? "" : d.date.slice(0, 4))
                        }
                        onAction={d.source ? undefined : () => setActing(d)}
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
