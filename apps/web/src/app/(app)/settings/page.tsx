"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { Main } from "@/components/layout/screen";
import { Button, Para, Section, Sheet, SettingRow, Skeleton, Title, cx } from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { logoutAction } from "@/features/auth/actions";
import { useCouple, useDeleteAccount } from "@/features/couple/hooks";
import { useDemoWeek, useWeek } from "@/features/prayers/hooks";
import { usePrefs } from "@/features/settings/hooks";
import { isMockApi } from "@/lib/config";
import { usePausedCount } from "@/lib/query/offline";
import { clearSignedInState } from "@/lib/query/persist";
import { time12 } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { useInstall } from "@/lib/pwa/install";

export default function SettingsPage() {
  const couple = useCouple(); const prefs = usePrefs(); const install = useInstall(); const week = useWeek(); const demo = useDemoWeek(); const del = useDeleteAccount();
  const qc = useQueryClient();
  const [confirm, setConfirm] = useState<"logout" | "delete" | null>(null);
  const partner = couple.data?.partner?.display_name;
  // Changes still waiting for the connection are dropped on logout (they
  // belong to this user, on this device), so the confirmation says so.
  const unsynced = usePausedCount();
  const logout = async () => { clearSignedInState(qc); await logoutAction(); };
  const deleteAccount = () => del.mutate(undefined, { onSuccess: async () => { clearSignedInState(qc); await logoutAction(); } });
  return (
    <>
      <Main>
        <div className="pt-1.5"><Title>Settings</Title></div>
        <QueryState queries={[couple, prefs]} loading={<Skeleton lines={4} />}>
          {(coupleData, prefsData) => {
            const me = coupleData.me;
            const mode = week.data ? (week.data.status === "waiting" ? "waiting" : week.data.setter_id === me.id ? "mine" : "partner") : "partner";
            return (
              <>
                <Section label={partner ? `You and ${partner}` : "You"} className="mt-[26px]">
                  <SettingRow icon="user" label="Profile" value={me.display_name} href={routes.settingsProfile} />
                  <SettingRow icon="users" label="Couple" value={partner ? `With ${partner}` : "Waiting for your partner"} href={partner ? routes.settingsProfile : routes.invite} last />
                </Section>
                <Section label="Reminders" className="mt-[26px]">
                  <SettingRow icon="bell" label="Notifications" value={Object.values(prefsData).some((v) => v === true) ? "On" : "Off"} href={routes.settingsNotifications} />
                  <SettingRow icon="clock" label="Prayer reminder" value={prefsData.prayer_reminder ? time12(prefsData.reminder_time) : "Off"} href={routes.settingsNotifications} />
                  <SettingRow icon="globe" label="Timezone" value={me.timezone === "Africa/Lagos" ? "Lagos" : me.timezone} href={routes.settingsProfile} last />
                </Section>
                <Section label="App" className="mt-[26px]">
                  <SettingRow icon="phone" label="Install Amorae" value={install.standalone ? "Installed" : "Not yet"} href={routes.install} />
                  <SettingRow icon="lock" label="Privacy" value="Coming soon" chevron={false} last />
                </Section>
                {isMockApi ? (
                  <Section label="Try the week states" className="mt-[26px]">
                    <div role="radiogroup" aria-label="Demo week state" className="flex gap-1 py-1">
                      {([["partner", "Praying"], ["mine", "Your week"], ["waiting", "Waiting"]] as const).map(([v, l]) => <button key={v} type="button" role="radio" aria-checked={mode === v} onClick={() => demo.mutate(v)} className={cx("press h-9 rounded-full px-3.5 text-[14px] font-semibold", mode === v ? "bg-plum-tint text-plum" : "text-stone")}>{l}</button>)}
                    </div>
                    <Para size="support">Mock mode only. The Go scheduler decides this for real, every Sunday.</Para>
                  </Section>
                ) : null}
                <Section label="Account" className="mt-[26px]">
                  <SettingRow icon="user" label="Account" value={me.email} href={routes.settingsProfile} />
                  <SettingRow icon="logout" label="Log out" chevron={false} onClick={() => setConfirm("logout")} />
                  <SettingRow icon="trash" label="Delete account" tone="red" chevron={false} onClick={() => setConfirm("delete")} last />
                </Section>
              </>
            );
          }}
        </QueryState>
        <div className="pb-2 pt-[22px] text-[13px] text-stone">Amorae 1.0 &middot; Two hearts, one faith.</div>
      </Main>
      <Sheet open={confirm === "logout"} onClose={() => setConfirm(null)} title="Log out of Amorae?" labelledBy="lo-h">
        {unsynced > 0 ? (
          <Para className="text-red">
            {unsynced === 1 ? "1 change hasn’t" : `${unsynced} changes haven’t`} synced yet. If you log out now, {unsynced === 1 ? "it" : "they"}&rsquo;ll be lost. Reconnect first to keep {unsynced === 1 ? "it" : "them"}.
          </Para>
        ) : (
          <Para>Your prayers and everything you&rsquo;ve shared stay safe. You&rsquo;ll just need to log in again.</Para>
        )}
        <div className="flex flex-col gap-1 pt-1"><Button onClick={logout}>Log out</Button><Button variant="text" onClick={() => setConfirm(null)}>Stay logged in</Button></div>
      </Sheet>
      <Sheet open={confirm === "delete"} onClose={() => setConfirm(null)} title="Delete your account?" labelledBy="del-h">
        <Para>This removes you from your space with {partner ?? "your partner"} and deletes everything you wrote. It can&rsquo;t be undone.</Para>
        <div className="flex flex-col gap-1 pt-1"><Button variant="secondary" className="border-red text-red" onClick={deleteAccount} loading={del.isPending}>Delete my account</Button><Button variant="text" onClick={() => setConfirm(null)}>Keep my account</Button></div>
      </Sheet>
    </>
  );
}
