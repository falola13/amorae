"use client";

import { useState } from "react";

import { useCouple } from "@/features/couple/hooks";
import { usePrefs, useSavePrefs } from "@/features/settings/hooks";
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

export default function NotificationSettings() {
  const prefs = usePrefs();
  const save = useSavePrefs();
  const couple = useCouple();
  const push = usePush();
  const [asking, setAsking] = useState(false);
  // usePush explains nothing by itself; the copy above this button is the
  // explaining, which is why the request lives here and not on a switch.
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
            ) : push.state === "default" ? (
              /* The switches below save either way, but nothing can arrive
                 until the browser has been asked — and it is only asked once,
                 so it is asked here with a reason rather than on a stray tap
                 (FR-NOTF-006). */
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
              {/* This row had its own hand-rolled invisible input and never
                  got the fix the compose rows did, so it opened once and then
                  never again — the reminder time was, in practice, unchangeable
                  after the first go. It is a PickRow like every other time
                  field now. */}
              <PickRow
                icon="clock"
                label="Reminder time"
                value={p.reminder_time}
                type="time"
                disabled={!p.prayer_reminder}
                onChange={(v: string) => v && save.mutate({ reminder_time: v })}
              />
              {/* In Faith rather than Together: it is about the prayers, and
                  it is the one notification in here that carries good news
                  rather than a reminder. */}
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
              {row(p, "challenges", "Challenge reminders", true)}
            </Section>
            <Para size="support" className="mb-4 mt-4">
              One prayer reminder a day, and one before anything you&rsquo;ve planned.
            </Para>
          </Main>
        )}
      </QueryState>
    </>
  );
}
