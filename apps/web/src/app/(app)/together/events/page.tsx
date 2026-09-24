"use client";

import { useEvents } from "@/features/together/hooks";
import { iso, relativeDay, stillAhead, time12 } from "@/lib/dates";
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
  // The empty state already offers this, centred, with a line saying what
  // it is for. Showing the bar as well put two buttons for the same thing
  // on one screen, one under the other. The bar is for when there is a
  // list to add to.
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
            // "Coming up" means still to come, not "today or later": an
            // event at 3:15am is over by breakfast, and leaving it under
            // Coming up all day is the app telling you something it can see
            // is untrue.
            const now = today();
            const up = eventsData
              .filter((e) => !e.done && stillAhead(e.date, e.start_time, now))
              .sort((a, b) =>
                (a.date + (a.start_time ?? "")).localeCompare(b.date + (b.start_time ?? "")),
              );
            const past = eventsData
              .filter((e) => e.done || !stillAhead(e.date, e.start_time, now))
              .sort((a, b) =>
                (b.date + (b.start_time ?? "")).localeCompare(a.date + (a.start_time ?? "")),
              );
            return (
              <>
                {up.length ? (
                  <Section label="Coming up" className="mt-5">
                    {up.map((e) => (
                      <DateRow
                        key={e.id}
                        date={e.date}
                        title={e.title}
                        sub={`${relativeDay(e.date, todayIso)}${e.start_time ? `, ${time12(e.start_time)}` : ""}${e.date === todayIso && e.location ? ` · ${e.location}` : ""}`}
                        href={routes.event(e.id)}
                      />
                    ))}
                  </Section>
                ) : null}
                {past.length ? (
                  <Section label="Earlier" className="mb-4 mt-6">
                    {past.map((e) => (
                      <DateRow
                        key={e.id}
                        date={e.date}
                        title={e.title}
                        // "Done together" used to be said of everything down
                        // here, because everything down here used to have
                        // been ticked. Now that an event also arrives by
                        // simply happening, saying it of one nobody marked is
                        // the app telling them they did something they may
                        // not have.
                        sub={
                          e.done
                            ? "Done together"
                            : `${relativeDay(e.date, todayIso)}${e.start_time ? `, ${time12(e.start_time)}` : ""}`
                        }
                        past
                        href={routes.event(e.id)}
                        action={e.done ? "Save a moment from it" : undefined}
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
          <LinkButton href={routes.eventNew()} icon="plus">
            Add an event
          </LinkButton>
        </BottomActions>
      ) : null}
    </>
  );
}
