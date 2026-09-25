"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { Main } from "@/components/layout/screen";
import {
  Button,
  Para,
  Section,
  Sheet,
  SettingRow,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { logoutAction } from "@/features/auth/actions";
import { DeleteAccountSheet } from "@/features/couple/components/delete-account-sheet";
import { useCouple, useEndedCouples, useMe } from "@/features/couple/hooks";
import { usePrefs } from "@/features/settings/hooks";
import { usePausedCount } from "@/lib/query/offline";
import { clearSignedInState } from "@/lib/query/persist";
import { longDate, time12 } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { useInstall } from "@/lib/pwa/install";

export default function SettingsPage() {
  // Reads the account rather than the couple, so it works before pairing and after leaving one.
  const me = useMe().data;
  const couple = useCouple();
  const ended = useEndedCouples();
  const prefs = usePrefs();
  const install = useInstall();
  const qc = useQueryClient();
  const [confirm, setConfirm] = useState<"logout" | "delete" | null>(null);
  const paired = Boolean(couple.data);
  const partner = couple.data?.partner?.display_name;
  // Surfaced near the top since it's time-limited (auto-deletes).
  const pastSpace = ended.data?.[0];
  // Unsynced local changes are dropped on logout, so the confirmation warns about it.
  const unsynced = usePausedCount();
  const logout = async () => {
    clearSignedInState(qc);
    await logoutAction();
  };
  return (
    <>
      {/* No tab bar without a couple, so this screen needs its own way back. */}
      {paired ? null : <TopBar back="Your space" backHref={routes.couple()} />}
      <Main>
        <div className="pt-1.5">
          <Title>Settings</Title>
        </div>
        {pastSpace ? (
          <Section label="Your past space" className="mt-[26px]">
            <SettingRow
              icon="clock"
              label={pastSpace.name}
              value={`Until ${longDate(pastSpace.read_only_until.slice(0, 10))}`}
              href={routes.settingsPastSpace}
              last
            />
          </Section>
        ) : null}
        {me ? (
          <Section label={partner ? `You and ${partner}` : "You"} className="mt-[26px]">
            <SettingRow
              icon="user"
              label="Profile"
              value={me.display_name}
              href={routes.settingsProfile}
            />
            <SettingRow
              icon="users"
              label="Couple"
              value={
                !paired
                  ? "Not started yet"
                  : partner
                    ? `With ${partner}`
                    : "Waiting for your partner"
              }
              href={paired ? routes.settingsCouple : routes.couple()}
              last
            />
          </Section>
        ) : (
          <div className="mt-[26px]">
            <Skeleton lines={2} />
          </div>
        )}
        <Section label="Reminders" className="mt-[26px]">
          <QueryState queries={[prefs]} loading={<Skeleton lines={2} />}>
            {(p) => (
              <>
                <SettingRow
                  icon="bell"
                  label="Notifications"
                  value={Object.values(p).some((v) => v === true) ? "On" : "Off"}
                  href={routes.settingsNotifications}
                />
                <SettingRow
                  icon="clock"
                  label="Prayer reminder"
                  value={p.prayer_reminder ? time12(p.reminder_time) : "Off"}
                  href={routes.settingsNotifications}
                />
              </>
            )}
          </QueryState>
          {me ? (
            <SettingRow
              icon="globe"
              label="Timezone"
              value={me.timezone === "Africa/Lagos" ? "Lagos" : me.timezone}
              href={routes.settingsProfile}
              last
            />
          ) : null}
        </Section>
        <Section label="App" className="mt-[26px]">
          <SettingRow
            icon="phone"
            label="Install Amorae"
            value={install.standalone ? "Installed" : "Not yet"}
            href={routes.install}
          />
          <SettingRow icon="lock" label="Privacy" value="Coming soon" chevron={false} last />
        </Section>
        <Section label="Account" className="mt-[26px]">
          {me ? (
            <SettingRow
              icon="user"
              label="Account"
              value={me.email}
              href={routes.settingsProfile}
            />
          ) : null}
          <SettingRow icon="lock" label="Where you're signed in" href={routes.settingsDevices} />
          <SettingRow
            icon="share"
            label="Download my data"
            href="/api/v1/users/me/export"
            download
            chevron={false}
          />
          <SettingRow
            icon="logout"
            label="Log out"
            chevron={false}
            onClick={() => setConfirm("logout")}
          />
          <SettingRow
            icon="trash"
            label="Delete account"
            tone="red"
            chevron={false}
            onClick={() => setConfirm("delete")}
            last
          />
        </Section>
        <div className="pb-2 pt-[22px] text-[13px] text-stone">
          Amorae 1.0 &middot; Two hearts, one faith.
        </div>
      </Main>
      <Sheet
        open={confirm === "logout"}
        onClose={() => setConfirm(null)}
        title="Log out of Amorae?"
        labelledBy="lo-h"
      >
        {unsynced > 0 ? (
          <Para className="text-red">
            {unsynced === 1 ? "1 change hasn’t" : `${unsynced} changes haven’t`} synced yet. If you
            log out now, {unsynced === 1 ? "it" : "they"}&rsquo;ll be lost. Reconnect first to keep{" "}
            {unsynced === 1 ? "it" : "them"}.
          </Para>
        ) : (
          <Para>
            Your prayers and everything you&rsquo;ve shared stay safe. You&rsquo;ll just need to log
            in again.
          </Para>
        )}
        <div className="flex flex-col gap-1 pt-1">
          <Button onClick={logout}>Log out</Button>
          <Button variant="text" onClick={() => setConfirm(null)}>
            Stay logged in
          </Button>
        </div>
      </Sheet>
      <DeleteAccountSheet
        open={confirm === "delete"}
        onClose={() => setConfirm(null)}
        partner={partner}
        onDeleted={logout}
      />
    </>
  );
}
