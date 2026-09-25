"use client";

import { useState } from "react";

import { useCouple } from "@/features/couple/hooks";
import { usePrefs, useSavePrefs } from "@/features/settings/hooks";
import { time12 } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { usePush } from "@/lib/pwa/push";
import type { NotificationPrefs } from "@/lib/api/types";
import { Main } from "@/components/layout/screen";
import {
  Button,
  LoadProblem,
  Para,
  Section,
  Skeleton,
  SwitchRow,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { PickRow } from "@/components/ui/pick-row";
import { QueryState, inPage } from "@/components/ui/query-state";

type BoolKey = {
  [K in keyof NotificationPrefs]: NotificationPrefs[K] extends boolean ? K : never;
}[keyof NotificationPrefs];

const capOptions = (max: number) => [
  { value: "0", label: "No limit" },
  ...Array.from({ length: max }, (_, i) => ({
    value: String(i + 1),
    label: `${i + 1} a day`,
  })),
];

function quietSentence(p: NotificationPrefs): string {
  const cap = p.daily_cap === 0 ? "As many as there are" : `At most ${p.daily_cap} a day`;
  if (!p.quiet_from || !p.quiet_to) return `${cap}, at any hour.`;
  return `${cap}, and none between ${time12(p.quiet_from)} and ${time12(p.quiet_to)}.`;
}

export default function NotificationSettings() {
  const prefs = usePrefs();
  const save = useSavePrefs();
  const couple = useCouple();
  const push = usePush();
  const [asking, setAsking] = useState(false);
  const allow = async () => {
    setAsking(true);
    try {
      await push.request();
    } finally {
      setAsking(false);
    }
  };
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
            ) : push.state === "default" ? (
              // Permission is requested here, with context, since the browser only asks once (FR-NOTF-006).
              <div className="mt-4 flex flex-col items-start gap-3 border-y border-line py-4">
                <div className="flex flex-col gap-1">
                  <div className="text-[16px] font-semibold text-ink">
                    Turn on notifications to receive these
                  </div>
                  <Para size="support">
                    Your choices below are saved either way — but until your browser allows them,
                    nothing can reach you.
                  </Para>
                </div>
                <Button variant="secondary" loading={asking} onClick={allow}>
                  Allow notifications
                </Button>
              </div>
            ) : null}
            <Section label="Faith" className="mt-[22px]">
              {row(p, "new_week", "New prayer week")}
              {row(p, "prayer_reminder", "Prayer reminder")}
              <PickRow
                icon="clock"
                label="Reminder time"
                value={p.reminder_time}
                type="time"
                disabled={!p.prayer_reminder}
                onChange={(v: string) => v && save.mutate({ reminder_time: v })}
              />
              {row(p, "prayer_answered", `When ${partner} marks a prayer answered`, true)}
            </Section>
            <Section label="Plans" className="mt-[22px]">
              {row(p, "event_reminders", "Event reminders")}
              {row(p, "important_dates", "Important dates", true)}
            </Section>
            <Section label="Together" className="mt-[22px]">
              {row(p, "appreciation", `Appreciation from ${partner}`)}
              {row(p, "journal", "Journal entries")}
              {row(p, "goals", "Goal updates")}
              {row(p, "challenges", "Challenge reminders")}
              {row(p, "together", "When you both finish something")}
              {row(p, "memories", "A moment from this day last year", true)}
            </Section>
            <Section label="How much" className="mt-[22px]">
              <PickRow
                icon="moon"
                label="Quiet from"
                value={p.quiet_from}
                type="time"
                onChange={(v: string) => v && save.mutate({ quiet_from: v })}
              />
              <PickRow
                icon="clock"
                label="Quiet until"
                value={p.quiet_to}
                type="time"
                onChange={(v: string) => v && save.mutate({ quiet_to: v })}
              />
              <PickRow
                icon="bell"
                label="Most in a day"
                value={String(p.daily_cap)}
                options={capOptions(p.max_daily_cap)}
                onChange={(v: string) => save.mutate({ daily_cap: Number(v) })}
                last
              />
            </Section>
            <Para size="support" className="mb-4 mt-4">
              {quietSentence(p)} Anything that can wait waits; a reminder whose moment has passed is
              dropped rather than saved up.
            </Para>
          </Main>
        )}
      </QueryState>
    </>
  );
}
