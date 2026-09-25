"use client";

import { useState } from "react";

import { Main } from "@/components/layout/screen";
import { Button, Micro, Para, Sheet, Skeleton, Title, TopBar } from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { useSessions, useSignOutOthers } from "@/features/settings/hooks";
import type { SessionInfo } from "@/lib/api/types";
import { longDate, timeAgo } from "@/lib/dates";
import { routes } from "@/lib/routes";

// Only this account's own sessions — never a partner's.
export default function Devices() {
  const sessions = useSessions();
  const signOutOthers = useSignOutOthers();
  const [confirm, setConfirm] = useState(false);

  const others = (sessions.data ?? []).filter((s) => !s.current).length;
  const endOthers = () => signOutOthers.mutate(undefined, { onSuccess: () => setConfirm(false) });

  return (
    <>
      <TopBar back="Settings" backHref={routes.settings} />
      <QueryState
        queries={[sessions]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton lines={3} />
          </Main>
        }
      >
        {(list) => (
          <Main>
            <div className="pt-3">
              <Title>Where you&rsquo;re signed in</Title>
              <Para className="mt-1.5">
                Every device signed in to your account. Signing out of the others ends them straight
                away; this one stays.
              </Para>
            </div>
            <div className="mt-6 flex flex-col border-t border-line">
              {list.map((session) => (
                <SessionRow key={session.created_at + session.device} session={session} />
              ))}
            </div>
            {others > 0 ? (
              <div className="mt-7">
                <Button
                  variant="secondary"
                  onClick={() => setConfirm(true)}
                  loading={signOutOthers.isPending}
                >
                  Sign out {others === 1 ? "the other device" : `the other ${others} devices`}
                </Button>
              </div>
            ) : (
              <Para size="support" className="mt-6">
                This is the only device signed in.
              </Para>
            )}
          </Main>
        )}
      </QueryState>
      <Sheet
        open={confirm}
        onClose={() => setConfirm(false)}
        title="Sign out other devices?"
        labelledBy="so-h"
      >
        <Para>
          {others === 1 ? "That device" : "Those devices"} will need your email and password again.
          You&rsquo;ll stay signed in here.
        </Para>
        <div className="flex flex-col gap-1 pt-1">
          <Button onClick={endOthers} loading={signOutOthers.isPending}>
            Sign {others === 1 ? "it" : "them"} out
          </Button>
          <Button variant="text" onClick={() => setConfirm(false)}>
            Keep {others === 1 ? "it" : "them"} signed in
          </Button>
        </div>
      </Sheet>
    </>
  );
}

function SessionRow({ session }: { session: SessionInfo }) {
  const used = session.last_used_at
    ? `Last used ${timeAgo(session.last_used_at)}`
    : "Not used since";
  return (
    <div className="flex min-h-[62px] flex-col justify-center gap-0.5 border-b border-line py-3">
      <div className="flex items-center gap-2">
        <span className="text-[16px] font-medium text-ink">{session.device}</span>
        {session.current ? <Micro tone="plum">This device</Micro> : null}
      </div>
      <span className="text-support text-stone">
        {used} &middot; signed in {longDate(session.created_at.slice(0, 10))}
      </span>
    </div>
  );
}
