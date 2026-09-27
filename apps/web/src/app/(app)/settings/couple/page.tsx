"use client";

import { useState } from "react";

import { Main } from "@/components/layout/screen";
import {
  cx,
  Initial,
  Micro,
  Para,
  Row,
  Section,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { EditCoupleSheet } from "@/features/couple/components/edit-couple-sheet";
import { LeaveCoupleSheet } from "@/features/couple/components/leave-couple-sheet";
import { useCouple } from "@/features/couple/hooks";
import { longDateYear, yearsAndMonths } from "@/lib/dates";
import { zoneLabel } from "@/lib/timezones";
import { routes } from "@/lib/routes";

// Shared couple settings only; a partner's own account (email, devices, sign-in) stays out of this screen (DEC-23).
export default function CouplePage() {
  const couple = useCouple();
  const [editing, setEditing] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const edit = () => setEditing(true);

  return (
    <>
      <TopBar back="Settings" backHref={routes.settings} />
      <QueryState
        queries={[couple]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton lines={4} />
          </Main>
        }
      >
        {(c) => (
          <Main>
            <div className="pt-3">
              <Title>{c.name}</Title>
              <Para className="pt-1.5">{blurb(c.started_on)}</Para>
            </div>

            <Section label="The two of you" className="pt-[26px]">
              <Person name={c.me.display_name} role={c.me.role} photoUrl={c.me.photo_url} you />
              {c.partner ? (
                <Person
                  name={c.partner.display_name}
                  role={c.partner.role}
                  photoUrl={c.partner.photo_url}
                  last
                />
              ) : (
                <Row
                  icon="users"
                  title="Your partner hasn’t joined yet"
                  sub="Send them the invitation"
                  href={routes.invite}
                  last
                />
              )}
            </Section>

            <Section label="Your space" className="pt-[26px]">
              <Row icon="together" title="Name" sub={c.name} onClick={edit} />
              <Row
                icon="calendar"
                title="Together since"
                sub={c.started_on ? longDateYear(c.started_on) : "Not set yet"}
                onClick={edit}
              />
              <Row
                icon="user"
                title="What you call yourself"
                sub={c.me.role || "Not set yet"}
                onClick={edit}
              />
              <Row
                icon="globe"
                title="Where your week starts"
                sub={zoneLabel(c.timezone) || "Not set yet"}
                onClick={edit}
                last
              />
            </Section>

            <Para size="support" className="pt-[18px]">
              What you call yourself is only a label, and only yours. It changes nothing about what
              either of you can do here.
            </Para>

            <Section label="Ending" className="pt-[30px]">
              <Row
                icon="logout"
                title="End this space"
                sub={
                  c.partner
                    ? `Ends it for ${c.partner.display_name} too`
                    : "You can start again whenever you want"
                }
                onClick={() => setLeaving(true)}
                last
              />
            </Section>
            <Para size="support" className="pt-3">
              Everything stays readable for 30 days afterwards, so you can both take a copy.
            </Para>
          </Main>
        )}
      </QueryState>
      {couple.data ? (
        <>
          <EditCoupleSheet open={editing} onClose={() => setEditing(false)} couple={couple.data} />
          <LeaveCoupleSheet
            open={leaving}
            onClose={() => setLeaving(false)}
            partner={couple.data.partner?.display_name}
          />
        </>
      ) : null}
    </>
  );
}

// Reimplements Row's layout by hand since this needs an initial avatar instead of an icon.
function Person({
  name,
  role,
  photoUrl,
  you,
  last,
}: {
  name: string;
  role?: string;
  photoUrl?: string | null;
  you?: boolean;
  last?: boolean;
}) {
  return (
    <div className={cx("flex min-h-[62px] items-center gap-3.5", !last && "border-b border-line")}>
      <Initial letter={name[0] ?? "?"} photoUrl={photoUrl} name={name} size={34} />
      <span className="flex min-w-0 grow flex-col gap-px">
        <span className="text-[16px] font-semibold text-ink">{name}</span>
        {role ? <span className="text-support text-stone">{role}</span> : null}
      </span>
      {you ? <Micro tone="plum">You</Micro> : null}
    </div>
  );
}

/** "Together 2 years, 7 months", or something true when the date isn't set. */
function blurb(startedOn?: string) {
  if (!startedOn) return "Everything the two of you are building, in one place.";
  const elapsed = yearsAndMonths(startedOn);
  return elapsed ? `Together ${elapsed}.` : `Together since ${longDateYear(startedOn)}.`;
}
