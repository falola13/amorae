"use client";

import type { ReactNode } from "react";

import { Initial, LinkButton, Section, Segments } from "@/components/ui/kit";
import { setterWord, todaysPoints } from "@/features/prayers/derive";
import { useTodayRequests } from "@/features/prayers/hooks";
import type { Couple, PrayerWeek } from "@/lib/api/types";
import { routes } from "@/lib/routes";
import { MyRequest, TheirRequest } from "./prayer-request-card";

/**
 * Everything to do with praying today, in one place: their request if they
 * asked, the week's progress and the one primary action, whether they have
 * prayed, and your own request as a quiet line underneath. Until the week is
 * shared it holds the prompt to share it instead of progress.
 *
 * `weekPrompt` is told whether it may be the raised leaf: it may not while
 * their unprayed request already is.
 */
export function TodaysPrayer({
  couple,
  week,
  weekPrompt,
  reminder,
}: {
  couple: Couple;
  week: PrayerWeek | undefined;
  weekPrompt: (raised: boolean) => ReactNode;
  reminder: string | null;
}) {
  const requests = useTodayRequests();
  const partner = couple.partner?.display_name ?? "your partner";
  const first = partner.split(" ")[0];
  const theirs = requests.data?.theirs ?? null;
  const mine = requests.data?.mine ?? null;
  const total = week?.status === "published" ? todaysPoints(week).length : 0;
  const done = week?.my_completed.length ?? 0;

  return (
    <Section
      label="Today’s prayer"
      className="mt-8"
      trailing={
        week?.status === "published" ? (
          <span className="lining text-support text-stone">
            {total > 0 ? `${done} of ${total} · ` : ""}set by{" "}
            {setterWord(week, couple.me.id, partner)}
          </span>
        ) : undefined
      }
    >
      {theirs ? <TheirRequest request={theirs} first={first} /> : null}

      {week?.status === "published" ? (
        <>
          {total > 0 ? (
            <div className="mt-4 flex flex-col gap-2">
              <Segments total={total} done={done} height={4} />
              <span className="lining text-support text-stone">
                today {done} of {total} prayed
              </span>
            </div>
          ) : null}
          <div className="mt-4">
            <LinkButton href={routes.prayerMode()}>
              {total === 0 || done >= total
                ? "Open prayer mode"
                : done === 0
                  ? "Begin praying"
                  : "Continue praying"}
            </LinkButton>
          </div>
          <div className="mt-3 flex items-center gap-2 text-support text-stone">
            <Initial
              letter={partner[0]}
              photoUrl={couple.partner?.photo_url}
              name={partner}
              size={22}
            />
            {total === 0
              ? "Nothing set for today"
              : week.partner_completed.length > 0
                ? `${first} has prayed today`
                : `${first} hasn’t prayed yet today`}
            {reminder ? <> &middot; reminder {reminder}</> : null}
          </div>
        </>
      ) : (
        weekPrompt(Boolean(requests.data) && !(theirs && !theirs.prayed_at))
      )}

      {requests.data ? <MyRequest request={mine} first={first} /> : null}
    </Section>
  );
}
