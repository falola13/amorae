"use client";

import Link from "next/link";
import { useCouple } from "@/features/couple/hooks";
import { DaysRow } from "@/features/prayers/components/days-row";
import { setterLabel, prayerCount } from "@/features/prayers/derive";
import { useHistory, useWeek } from "@/features/prayers/hooks";
import { longDate, monthName } from "@/lib/dates";
import { routes } from "@/lib/routes";
import type { PrayerWeek } from "@/lib/api/types";
import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import {
  EmptyState,
  LinkButton,
  Micro,
  Ornament,
  Para,
  Skeleton,
  Title,
  cx,
} from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";

function sinceLine(count: number, first?: string) {
  const weeks = count === 1 ? "One week" : `${count} weeks`;
  if (!first) return `${weeks} together.`;
  return `${weeks} together, since ${longDate(first)} ${first.slice(0, 4)}.`;
}

function Entry({
  w,
  now,
  last,
  partner,
  meId,
}: {
  w: PrayerWeek;
  now?: boolean;
  last?: boolean;
  partner: string;
  meId?: string;
}) {
  const meta = now
    ? `${prayerCount(w.points.length)} · you ${w.my_completed.length} of ${w.points.length} so far`
    : `${prayerCount(w.points.length)} · you ${w.my_completed.length} of ${w.points.length} · ${partner} ${w.partner_completed.length} of ${w.points.length}`;
  return (
    <Link
      href={now ? routes.prayers : routes.historyWeek(w.id)}
      className="press relative flex min-h-[84px] items-center gap-3 pl-[30px] text-ink no-underline"
    >
      <span aria-hidden="true" className="absolute bottom-0 left-1 top-0 w-px bg-line" />
      <span
        aria-hidden="true"
        className={cx(
          "absolute left-0 top-[22px] box-border h-[9px] w-[9px] rounded-full",
          now ? "border-2 border-plum bg-plum" : "border-[1.5px] border-edge bg-bg",
        )}
      />
      <span className={cx("flex grow flex-col gap-1.5 py-3.5", !last && "border-b border-line")}>
        <span className="flex items-center gap-2.5 text-bodylg font-semibold tracking-[-0.01em]">
          {longDate(w.week_start)}
          {now ? (
            <span className="rounded-full bg-plum-tint px-2 py-0.5 text-[12px] font-bold text-plum">
              This week
            </span>
          ) : null}
        </span>
        <span className="flex flex-col gap-0.5">
          <span className="text-[15px]">{setterLabel(w, meId, partner)} set the prayers</span>
          <span className="text-support text-stone">{meta}</span>
        </span>
        {w.days.length > 0 ? (
          <div className="max-w-[220px] pt-0.5">
            <DaysRow days={w.days} today={w.today} partnerName={partner} />
          </div>
        ) : null}
      </span>
      <Icon name="right" size={18} className="text-stone" />
    </Link>
  );
}

export default function History() {
  const history = useHistory();
  const week = useWeek();
  const couple = useCouple();
  const partner = couple.data?.partner?.display_name ?? "Your partner";
  return (
    <Main>
      <div className="pt-1.5">
        <Title>History</Title>
      </div>
      <QueryState queries={[history]} loading={<Skeleton lines={4} />}>
        {(historyData) => {
          const all = [
            ...(week.data && week.data.status === "published" ? [week.data] : []),
            ...historyData,
          ];
          if (all.length === 0) {
            return (
              <EmptyState
                ghost="dots"
                title="Your first week will appear here"
                text="Each Sunday, the week you’ve just finished is kept here for the two of you to look back on."
                cta={<LinkButton href={routes.prayers}>Go to this week’s prayers</LinkButton>}
              />
            );
          }
          const groups: { month: string; weeks: PrayerWeek[] }[] = [];
          for (const w of all) {
            const m = monthName(w.week_start);
            const g = groups[groups.length - 1];
            if (g && g.month === m) g.weeks.push(w);
            else groups.push({ month: m, weeks: [w] });
          }
          return (
            <>
              <Para className="mt-1.5">Every prayer week the two of you have shared.</Para>
              <div className="mt-2 flex flex-col pb-4">
                {groups.map((g, gi) => (
                  <div key={g.month}>
                    <div className="pb-1.5 pl-[30px] pt-[18px]">
                      <Micro>{g.month}</Micro>
                    </div>
                    {g.weeks.map((w, i) => (
                      <Entry
                        key={w.id}
                        w={w}
                        now={w.id === week.data?.id}
                        partner={partner}
                        meId={couple.data?.me.id}
                        last={gi === groups.length - 1 && i === g.weeks.length - 1}
                      />
                    ))}
                  </div>
                ))}
                <Ornament caption={sinceLine(all.length, all[all.length - 1]?.week_start)} />
              </div>
            </>
          );
        }}
      </QueryState>
    </Main>
  );
}
