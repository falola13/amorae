"use client";

import Link from "next/link";
import { useEffect, useRef } from "react";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import { EmptyState, LinkButton, Micro, Para, Skeleton, Title, cx } from "@/components/ui/kit";
import { iconForKind, groupInbox } from "@/features/notifications/derive";
import { useInbox, useMarkAllRead } from "@/features/notifications/hooks";
import type { NotificationItem } from "@/lib/api/types";
import { iso, timeAgo } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";

function InboxRow({ item }: { item: NotificationItem }) {
  return (
    <Link
      href={item.path}
      className="press flex min-h-[64px] items-start gap-3.5 border-b border-line py-3 text-ink no-underline last:border-b-0"
    >
      <Icon name={iconForKind(item.kind)} size={22} className="mt-0.5 shrink-0 text-stone" />
      <span className="flex min-w-0 grow flex-col gap-0.5">
        <span className="flex items-center gap-1.5">
          {!item.read ? (
            <span aria-hidden="true" className="h-1.5 w-1.5 shrink-0 rounded-full bg-plum" />
          ) : null}
          <span className={cx("truncate text-[16px]", item.read ? "font-medium" : "font-bold")}>
            {item.title}
          </span>
        </span>
        <span className="truncate text-support text-stone">{item.body}</span>
      </span>
      <span className="shrink-0 whitespace-nowrap pl-1 text-support text-stone">
        {timeAgo(item.created_at)}
      </span>
    </Link>
  );
}

export default function Inbox() {
  const inbox = useInbox();
  const markAllRead = useMarkAllRead();
  const todayIso = iso(today());
  const firedRef = useRef(false);

  useEffect(() => {
    if (firedRef.current) return;
    if (!inbox.data || inbox.data.unread === 0) return;
    firedRef.current = true;
    markAllRead.mutate(undefined);
    // Runs once, the first time the inbox loads with anything unread; deliberately
    // not re-run on every refetch, which would fire the POST on every 60s poll.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [inbox.data]);

  return (
    <>
      <div className="flex h-11 shrink-0 items-center px-3">
        <Link
          href={routes.home}
          className="press flex h-11 items-center gap-0.5 pl-1 pr-2 text-[16px] font-medium text-ink no-underline"
        >
          <Icon name="left" size={22} />
          Home
        </Link>
      </div>
      <Main>
        <div className="pt-2">
          <Title>Notifications</Title>
          <Para className="mt-1.5">Your last 30, so nothing gets lost.</Para>
        </div>
        <QueryState queries={[inbox]} loading={<Skeleton />}>
          {(data) => {
            if (data.items.length === 0) {
              return (
                <EmptyState
                  ghost="dots"
                  title="Nothing yet."
                  text="When something happens between you, it lands here too."
                  cta={
                    <LinkButton href={routes.home} variant="secondary">
                      Back to home
                    </LinkButton>
                  }
                />
              );
            }
            const { today: todays, earlier } = groupInbox(data.items, todayIso);
            return (
              <div className="mt-4 flex flex-col gap-6">
                {todays.length ? (
                  <section>
                    <Micro className="pb-1">Today</Micro>
                    <div className="flex flex-col">
                      {todays.map((item) => (
                        <InboxRow key={item.id} item={item} />
                      ))}
                    </div>
                  </section>
                ) : null}
                {earlier.length ? (
                  <section>
                    <Micro className="pb-1">Earlier</Micro>
                    <div className="flex flex-col">
                      {earlier.map((item) => (
                        <InboxRow key={item.id} item={item} />
                      ))}
                    </div>
                  </section>
                ) : null}
              </div>
            );
          }}
        </QueryState>
      </Main>
    </>
  );
}
