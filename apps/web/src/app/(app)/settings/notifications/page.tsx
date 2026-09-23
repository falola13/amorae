"use client";

import { useCouple } from "@/features/couple/hooks";
import { usePrefs, useSavePrefs } from "@/features/settings/hooks";
import { time12 } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { usePush } from "@/lib/pwa/push";
import type { NotificationPrefs } from "@/lib/api/types";
import { Main } from "@/components/layout/screen";
import {
  LoadProblem,
  Para,
  Section,
  Skeleton,
  SwitchRow,
  Title,
  TopBar,
  cx,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";

type BoolKey = {
  [K in keyof NotificationPrefs]: NotificationPrefs[K] extends boolean ? K : never;
}[keyof NotificationPrefs];

export default function NotificationSettings() {
  const prefs = usePrefs();
  const save = useSavePrefs();
  const couple = useCouple();
  const push = usePush();
  const partner = couple.data?.partner?.display_name ?? "your partner";
  const row = (p: NotificationPrefs, k: BoolKey, label: string, last?: boolean) => (
    <SwitchRow
      key={k}
      label={label}
      last={last}
      checked={p[k]}
      onChange={(v) => save.mutate({ [k]: v })}
    />
  );
  return (
    <>
      <TopBar back="Settings" backHref={routes.settings} />
      <QueryState
        queries={[prefs]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(p) => (
          <Main>
            <div className="pt-3">
              <Title>Notifications</Title>
              <Para className="mt-1.5">Few, calm, and only about the two of you.</Para>
            </div>
            {/* Switches that can't deliver anything would be a lie: if the
                browser permission is denied or unsupported, say so first. */}
            {push.state === "denied" ? (
              <LoadProblem
                tone="quiet"
                icon="bell"
                title="Notifications are blocked"
                text="Your browser is blocking them for Amorae. Allow notifications in its settings, then come back — your choices below are saved either way."
              />
            ) : push.state === "unsupported" ? (
              <LoadProblem
                tone="quiet"
                icon="bell"
                title="This browser can't show notifications"
                text="Install Amorae to your Home Screen, or open it in another browser. Your choices below are saved either way."
              />
            ) : null}
            <Section label="Faith" className="mt-[22px]">
              {row(p, "new_week", "New prayer week")}
              {row(p, "prayer_reminder", "Prayer reminder")}
              <label
                className={cx(
                  "relative flex h-[54px] items-center justify-between pl-4 text-[16px] font-medium",
                  !p.prayer_reminder && "opacity-50",
                )}
              >
                Reminder time
                <span className="tabular flex items-center gap-1.5 font-semibold text-plum">
                  {time12(p.reminder_time)}, every day
                </span>
                <input
                  type="time"
                  value={p.reminder_time}
                  disabled={!p.prayer_reminder}
                  onChange={(e) => save.mutate({ reminder_time: e.target.value })}
                  className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
                  aria-label="Reminder time"
                />
              </label>
            </Section>
            <Section label="Plans" className="mt-[22px]">
              {row(p, "event_reminders", "Event reminders")}
              {row(p, "important_dates", "Important dates", true)}
            </Section>
            <Section label="Together" className="mt-[22px]">
              {row(p, "appreciation", `Appreciation from ${partner}`)}
              {row(p, "journal", "Journal entries")}
              {row(p, "goals", "Goal updates")}
              {row(p, "challenges", "Challenge reminders", true)}
            </Section>
            <Para size="support" className="mb-4 mt-4">
              Never more than one reminder a day.
            </Para>
          </Main>
        )}
      </QueryState>
    </>
  );
}
