"use client";

import { useCouple } from "@/features/couple/hooks";
import { eventOwnerLabel, eventPhase, nowLabel } from "@/features/together/events";
import { useEvents } from "@/features/together/hooks";
import { iso, relativeDay, time12 } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { DateRow } from "@/components/ui/date-row";
import { today } from "@/lib/today";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import {
  BottomActions,
  EmptyState,
  LinkButton,
  Section,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";

export default function Events() {
  const events = useEvents();
  const couple = useCouple();
  const meId = couple.data?.me.id;
  const partnerName = couple.data?.partner?.display_name;
  // Hides the bottom add-button when the empty state already shows one.
  const hasAny = (events.data?.length ?? 0) > 0;
  const todayIso = iso(today());
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="pt-2">
          <Title>Events</Title>
        </div>
        <QueryState queries={[events]} loading={<Skeleton />}>
          {(eventsData) => {
            if (eventsData.length === 0) {
              return (
                <EmptyState
                  ghost="lines"
                  title="Nothing planned yet."
                  text="Add something you’d love to do together."
                  cta={
                    <LinkButton href={routes.eventNew()} icon="plus">
                      Add an event
                    </LinkButton>
                  }
                />
              );
            }
            // One rule (eventPhase) decides the section, so an event moves on as its time does.
            const now = today();
            const byStart = (x: { date: string; start_time?: string }) =>
              x.date + (x.start_time ?? "");
            const phased = eventsData.map((e) => ({ e, phase: eventPhase(e, now) }));
            const ongoing = phased
              .filter((x) => x.phase === "ongoing")
              .map((x) => x.e)
              .sort((a, b) => byStart(a).localeCompare(byStart(b)));
            const up = phased
              .filter((x) => x.phase === "upcoming")
              .map((x) => x.e)
              .sort((a, b) => byStart(a).localeCompare(byStart(b)));
            const past = phased
              .filter((x) => x.phase === "over")
              .map((x) => x.e)
              .sort((a, b) => byStart(b).localeCompare(byStart(a)));
            return (
              <>
                {ongoing.length ? (
                  <Section label="Happening now" className="mt-5">
                    {ongoing.map((e) => {
                      const owner = eventOwnerLabel(e, meId, partnerName);
                      const base = `${nowLabel(e)}${e.location ? ` · ${e.location}` : ""}`;
                      return (
                        <DateRow
                          key={e.id}
                          date={e.date}
                          title={e.title}
                          sub={owner ? `${base} · ${owner}` : base}
                          href={routes.event(e.id)}
                        />
                      );
                    })}
                  </Section>
                ) : null}
                {up.length ? (
                  <Section label="Coming up" className="mt-5">
                    {up.map((e) => {
                      const owner = eventOwnerLabel(e, meId, partnerName);
                      return (
                        <DateRow
                          key={e.id}
                          date={e.date}
                          title={e.title}
                          sub={`${relativeDay(e.date, todayIso)}${e.start_time ? `, ${time12(e.start_time)}` : ""}${e.date === todayIso && e.location ? ` · ${e.location}` : ""}${owner ? ` · ${owner}` : ""}`}
                          href={routes.event(e.id)}
                        />
                      );
                    })}
                  </Section>
                ) : null}
                {past.length ? (
                  <Section label="Earlier" className="mb-4 mt-6">
                    {past.map((e) => {
                      const owner = eventOwnerLabel(e, meId, partnerName);
                      const base = e.didnt_happen
                        ? "Didn’t happen"
                        : e.done
                          ? "Done"
                          : `${relativeDay(e.date, todayIso)}${e.start_time ? `, ${time12(e.start_time)}` : ""}`;
                      return (
                        <DateRow
                          key={e.id}
                          date={e.date}
                          title={e.title}
                          sub={owner ? `${base} · ${owner}` : base}
                          past
                          href={routes.event(e.id)}
                        />
                      );
                    })}
                  </Section>
                ) : null}
              </>
            );
          }}
        </QueryState>
      </Main>
      {hasAny ? (
        <BottomActions>
          <LinkButton href={routes.eventNew()} icon="plus">
            Add an event
          </LinkButton>
        </BottomActions>
      ) : null}
    </>
  );
}
