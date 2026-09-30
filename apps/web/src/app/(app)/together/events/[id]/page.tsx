"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import {
  useDeleteEvent,
  useEventOutcome,
  useEvent,
  useChecklist,
  usePatchEvent,
} from "@/features/together/hooks";
import {
  canManageEvent,
  eventAddedByLabel,
  eventOwnerLabel,
  eventPhase,
  nowLabel,
} from "@/features/together/events";
import { KeepAsMemorySheet } from "@/features/together/components/keep-as-memory-sheet";
import { useCouple } from "@/features/couple/hooks";
import { RemindersEditor } from "@/features/together/components/reminders-editor";
import { today } from "@/lib/today";
import { Icon } from "@/components/icons";
import { PickRow } from "@/components/ui/pick-row";
import type { Event } from "@/lib/api/types";
import type { EventInput } from "@/lib/api/schemas";
import { weekdayDate } from "@/lib/dates";
import type { EventOutcome } from "@/features/together/api";
import { routes } from "@/lib/routes";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import {
  BottomActions,
  Button,
  Checkbox,
  Micro,
  Para,
  Section,
  Sheet,
  Skeleton,
  Title,
  TopBar,
  cx,
} from "@/components/ui/kit";

// Each row is its own control and PATCHes only the field it owns, leaving the rest of the event untouched.
// canEdit false (a partner's own "mine" event): the rows show their values but take no input.
function When({ event, canEdit }: { event: Event; canEdit: boolean }) {
  const patch = usePatchEvent();
  const set = (field: keyof EventInput) => (value: string) =>
    patch.mutate({ id: event.id, patch: { [field]: value } });

  return (
    <div className="mt-5 flex flex-col border-t border-line">
      <PickRow
        icon="clock"
        label="Starts"
        value={event.start_time ?? ""}
        type="time"
        onChange={canEdit ? set("start_time") : undefined}
        placeholder="Add a time"
      />
      <PickRow
        icon="clock"
        label="Ends"
        value={event.end_time ?? ""}
        type="time"
        onChange={canEdit ? set("end_time") : undefined}
        placeholder="Add a time"
      />
      <PickRow
        icon="pin"
        label="Location"
        value={event.location ?? ""}
        type="text"
        onChange={canEdit ? set("location") : undefined}
        placeholder="Add a place"
      />
      <RemindersEditor
        value={event.reminders ?? []}
        onChange={
          canEdit ? (next) => patch.mutate({ id: event.id, patch: { reminders: next } }) : undefined
        }
      />
    </div>
  );
}

/** A checklist item that can't be ticked — the partner's view of someone else's "mine" event. */
function ReadOnlyCheck({ checked, children }: { checked: boolean; children: React.ReactNode }) {
  return (
    <div
      className={cx(
        "flex h-12 w-full items-center gap-3.5 text-[16px] font-medium",
        checked ? "text-stone" : "text-ink",
      )}
    >
      {checked ? (
        <span className="flex h-[22px] w-[22px] shrink-0 items-center justify-center rounded-[7px] bg-green-tint">
          <Icon name="check" size={14} strokeWidth={2.2} className="text-green" />
        </span>
      ) : (
        <span className="box-border h-[22px] w-[22px] shrink-0 rounded-[7px] border-[1.5px] border-edge" />
      )}
      {children}
    </div>
  );
}

export default function EventDetail() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const ev = useEvent(id);
  const toggle = useChecklist();
  const outcome = useEventOutcome();
  const remove = useDeleteEvent();
  const couple = useCouple();
  const meId = couple.data?.me.id;
  const partnerName = couple.data?.partner?.display_name;
  const [confirming, setConfirming] = useState(false);
  const [keeping, setKeeping] = useState(false);
  const canManage = ev.data ? canManageEvent(ev.data, meId) : true;
  const edit =
    ev.data && canManage ? (
      <Link
        href={routes.eventNew({ edit: ev.data.id })}
        className="press flex h-11 items-center px-3 text-[16px] font-semibold text-plum no-underline"
      >
        Edit
      </Link>
    ) : undefined;
  return (
    <>
      <TopBar back="Events" backHref={routes.events} right={edit} />
      <QueryState
        queries={[ev]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(e) => {
          const phase = eventPhase(e, today());
          const over = phase === "over";
          const ongoing = phase === "ongoing";
          const canEdit = canManageEvent(e, meId);
          // Leaving for the list only where the event drops out of view (marking it done from the page).
          const setOutcome = (o: EventOutcome, leave = false) =>
            outcome.mutate(
              { id: e.id, outcome: o },
              leave
                ? {
                    onSuccess: () => router.replace(routes.events),
                    onQueued: () => router.replace(routes.events),
                  }
                : undefined,
            );
          const ownerLabel = eventOwnerLabel(e, meId, partnerName);
          const addedByLabel = eventAddedByLabel(e, meId, partnerName);
          return (
            <>
              <Main>
                <div className="pt-3">
                  <Micro>{weekdayDate(e.date)}</Micro>
                </div>
                <Title size="lg" className="mt-2">
                  {e.title}
                </Title>
                {ongoing ? (
                  <Para size="support" className="mt-1 font-semibold text-plum">
                    Happening now{e.end_time ? ` · ${nowLabel(e).replace("Now · ", "")}` : ""}
                  </Para>
                ) : null}
                {e.didnt_happen ? (
                  <Para size="support" className="mt-1">
                    Marked as didn&rsquo;t happen.
                  </Para>
                ) : null}
                {ownerLabel || addedByLabel ? (
                  <Para size="support" className="mt-0.5">
                    {ownerLabel ?? addedByLabel}
                  </Para>
                ) : null}
                <When event={e} canEdit={canEdit} />
                {e.checklist.length ? (
                  <Section label="Before we go" className="mt-[22px]">
                    {e.checklist.map((c) =>
                      canEdit ? (
                        <Checkbox
                          key={c.id}
                          checked={c.done}
                          onChange={(v) => toggle.mutate({ id: e.id, item: c.id, done: v })}
                        >
                          {c.text}
                        </Checkbox>
                      ) : (
                        <ReadOnlyCheck key={c.id} checked={c.done}>
                          {c.text}
                        </ReadOnlyCheck>
                      ),
                    )}
                  </Section>
                ) : null}
                {e.notes ? (
                  <section className="mt-[18px] flex flex-col gap-1.5">
                    <Micro>Notes</Micro>
                    <p className="m-0 text-body" data-selectable>
                      {e.notes}
                    </p>
                  </section>
                ) : null}
              </Main>
              {canEdit ? (
                <BottomActions>
                  {/* Before it's over: mark it done. After: say how it went. */}
                  {!e.done && !e.didnt_happen && !over ? (
                    <Button
                      variant="secondary"
                      icon="check"
                      onClick={() => setOutcome("happened", true)}
                    >
                      Mark as done
                    </Button>
                  ) : null}
                  {!e.done && !e.didnt_happen && over ? (
                    <>
                      <Button icon="check" onClick={() => setOutcome("happened", true)}>
                        We did it
                      </Button>
                      <Button variant="secondary" icon="image" onClick={() => setKeeping(true)}>
                        Keep as a memory
                      </Button>
                      <Button variant="text" onClick={() => setOutcome("didnt_happen", true)}>
                        It didn&rsquo;t happen
                      </Button>
                    </>
                  ) : null}
                  {e.done ? (
                    <>
                      <Button variant="secondary" icon="image" onClick={() => setKeeping(true)}>
                        Keep it as a memory
                      </Button>
                      <Button variant="text" onClick={() => setOutcome("none")}>
                        Not done after all
                      </Button>
                    </>
                  ) : null}
                  {e.didnt_happen ? (
                    <Button variant="text" onClick={() => setOutcome("happened")}>
                      It happened after all
                    </Button>
                  ) : null}
                  <Button variant="text" className="text-red" onClick={() => setConfirming(true)}>
                    Delete this event
                  </Button>
                </BottomActions>
              ) : null}
              <KeepAsMemorySheet
                event={e}
                open={keeping}
                onClose={() => setKeeping(false)}
                onKept={() => router.push(routes.memories)}
              />
              <Sheet
                open={confirming}
                onClose={() => setConfirming(false)}
                title="Delete this event?"
                labelledBy="del-event-h"
              >
                <Para>
                  It disappears for both of you, along with anything on its checklist. You can
                  always plan it again.
                </Para>
                <div className="flex flex-col gap-1">
                  <Button
                    variant="secondary"
                    className="border-red text-red"
                    loading={remove.isPending}
                    onClick={() =>
                      remove.mutate(e.id, {
                        onSuccess: () => router.replace(routes.events),
                        onQueued: () => router.replace(routes.events),
                      })
                    }
                  >
                    Delete it
                  </Button>
                  <Button variant="text" onClick={() => setConfirming(false)}>
                    Keep it
                  </Button>
                </div>
              </Sheet>
            </>
          );
        }}
      </QueryState>
    </>
  );
}
