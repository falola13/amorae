"use client";

import { Main } from "@/components/layout/screen";
import { Icon } from "@/components/icons";
import {
  buttonClass,
  EmptyState,
  Initial,
  LinkButton,
  Para,
  Row,
  Section,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { useCouple, useEndedCouples } from "@/features/couple/hooks";
import { daysUntil, longDateYear, yearsAndMonths } from "@/lib/dates";
import type { EndedCouple } from "@/lib/api/types";
import { routes } from "@/lib/routes";

/**
 * A space that has ended, for as long as it is still here.
 *
 * Read-only by construction rather than by disabling things: there is nothing
 * on this screen to change. The one action is taking a copy, because that is
 * the only thing left that can still be lost.
 */
export default function PastSpacePage() {
  const ended = useEndedCouples();
  // Reachable both ways: from Settings once you have a space again, and
  // straight after ending one, when you have none and Settings would only
  // bounce you back to pairing.
  const couple = useCouple();
  const paired = Boolean(couple.data);

  return (
    <>
      {paired ? (
        <TopBar back="Settings" backHref={routes.settings} />
      ) : (
        <TopBar back="Start again" backHref={routes.couple()} />
      )}
      <QueryState
        queries={[ended]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton lines={4} />
          </Main>
        }
      >
        {(spaces) =>
          spaces.length === 0 ? (
            <Main>
              <div className="pt-6">
                <EmptyState
                  ghost="frames"
                  title="Nothing here"
                  text="When a space ends, it stays here for 30 days so you can read it and take a copy."
                  cta={
                    <LinkButton href={routes.settings} variant="secondary">
                      Back to settings
                    </LinkButton>
                  }
                />
              </div>
            </Main>
          ) : (
            <Main>
              {spaces.map((space) => (
                <PastSpace key={space.id} space={space} />
              ))}
              {!paired ? (
                <div className="pt-6">
                  <LinkButton href={routes.couple()}>Start a new space</LinkButton>
                </div>
              ) : null}
            </Main>
          )
        }
      </QueryState>
    </>
  );
}

function PastSpace({ space }: { space: EndedCouple }) {
  const endedOn = space.dissolved_at.slice(0, 10);
  const deletedOn = space.read_only_until.slice(0, 10);
  const daysLeft = daysUntil(deletedOn);

  return (
    <>
      <div className="pt-3">
        <Title>{space.name}</Title>
        <Para className="pt-1.5">
          Ended {longDateYear(endedOn)}
          {space.started_on
            ? ` · together ${yearsAndMonths(space.started_on, new Date(endedOn))}`
            : ""}
        </Para>
      </div>

      <Section label="The two of you" className="pt-[26px]">
        {space.people.map((person, i) => (
          <div
            key={person.id}
            className={
              "flex min-h-[62px] items-center gap-3.5" +
              (i === space.people.length - 1 ? "" : " border-b border-line")
            }
          >
            <Initial letter={person.display_name[0] ?? "?"} size={34} />
            <span className="flex min-w-0 grow flex-col gap-px">
              <span className="text-[16px] font-semibold text-ink">{person.display_name}</span>
              {person.role ? <span className="text-support text-stone">{person.role}</span> : null}
            </span>
          </div>
        ))}
      </Section>

      <Section label="Before it goes" className="pt-[26px]">
        <Row
          icon="clock"
          title={countdown(daysLeft)}
          sub={`Deleted on ${longDateYear(deletedOn)}`}
          last
        />
      </Section>

      <Para size="support" className="pt-3">
        Everything the two of you wrote here is deleted then, for both of you. A copy is yours to
        keep.
      </Para>

      <div className="pt-5">
        {/* A plain anchor, not LinkButton: a client-side navigation never saves a file. */}
        <a href="/api/v1/users/me/export" download className={buttonClass("secondary")}>
          <Icon name="share" size={20} />
          Download a copy
        </a>
      </div>
    </>
  );
}

function countdown(days: number): string {
  if (days <= 0) return "Going today";
  if (days === 1) return "1 day left";
  return `${days} days left`;
}
