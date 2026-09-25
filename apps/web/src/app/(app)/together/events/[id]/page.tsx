"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useState } from "react";
import {
  useCompleteEvent,
  useDeleteEvent,
  useEvent,
  useChecklist,
  usePatchEvent,
} from "@/features/together/hooks";
import { REMINDER_OPTIONS } from "@/features/together/events";
import { KeepAsMemorySheet } from "@/features/together/components/keep-as-memory-sheet";
import { stillAhead } from "@/lib/dates";
import { today } from "@/lib/today";
import { PickRow } from "@/components/ui/pick-row";
import type { Event } from "@/lib/api/types";
import type { EventInput } from "@/lib/api/schemas";
import { weekdayDate } from "@/lib/dates";
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
} from "@/components/ui/kit";

/**
 * The when and where of an event, changed from the event itself.
 *
 * These were read-only lines with a "Change" link on the reminder alone, so
 * tapping the time or the place did nothing at all — and a reminder took a
 * round trip through the whole compose form to move by an hour. Each row is
 * now the control, and each one saves on its own: the endpoint is a PATCH, so
 * a row sends the single field it owns and leaves the rest of the event,
 * checklist included, exactly where it was.
 *
 * Empty rows still show. "Add a time" is how you discover you can.
 */
function When({ event }: { event: Event }) {
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
        onChange={set("start_time")}
        placeholder="Add a time"
      />
      <PickRow
        icon="clock"
        label="Ends"
        value={event.end_time ?? ""}
        type="time"
        onChange={set("end_time")}
        placeholder="Add a time"
      />
      <PickRow
        icon="pin"
        label="Location"
        value={event.location ?? ""}
        type="text"
        onChange={set("location")}
        placeholder="Add a place"
      />
      <PickRow
        icon="bell"
        label="Reminder"
        value={event.reminder ?? ""}
        options={REMINDER_OPTIONS}
        onChange={set("reminder")}
        placeholder="None"
        last
      />
    </div>
  );
}

export default function EventDetail() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const ev = useEvent(id);
  const toggle = useChecklist();
  const complete = useCompleteEvent();
  const remove = useDeleteEvent();
  const [confirming, setConfirming] = useState(false);
  const [keeping, setKeeping] = useState(false);
  const edit = ev.data ? (
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
          // "Over" is a fact about the clock, not a thing to be told.
          const over = !stillAhead(e.date, e.start_time, today());
          return (
            <>
              <Main>
                <div className="pt-3">
                  <Micro>{weekdayDate(e.date)}</Micro>
                </div>
                <Title size="lg" className="mt-2">
                  {e.title}
                </Title>
                <When event={e} />
                {e.checklist.length ? (
                  <Section label="Before we go" className="mt-[22px]">
                    {e.checklist.map((c) => (
                      <Checkbox
                        key={c.id}
                        checked={c.done}
                        onChange={(v) => toggle.mutate({ id: e.id, item: c.id, done: v })}
                      >
                        {c.text}
                      </Checkbox>
                    ))}
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
              <BottomActions>
                {/* Once it is over, the app already knows it is over — asking
                  somebody to "mark as done" is asking them to confirm the
                  date. The question worth asking is the other one: did this
                  happen, and is it worth keeping. Before it happens, there is
                  nothing to ask at all, so the action stays out of the way. */}
                {e.done || over ? (
                  <Button variant="secondary" icon="image" onClick={() => setKeeping(true)}>
                    {e.done ? "Keep it as a memory" : "Keep this as a memory"}
                  </Button>
                ) : null}
                {!e.done && over ? (
                  <Button
                    variant="text"
                    onClick={() =>
                      complete.mutate(
                        { id: e.id, done: true },
                        { onSuccess: () => router.replace(routes.events) },
                      )
                    }
                  >
                    It didn&rsquo;t happen
                  </Button>
                ) : null}
                {!e.done && !over ? (
                  <Button
                    variant="secondary"
                    icon="check"
                    onClick={() =>
                      complete.mutate(
                        { id: e.id, done: true },
                        { onSuccess: () => router.replace(routes.events) },
                      )
                    }
                  >
                    Mark as done
                  </Button>
                ) : null}
                {e.done ? (
                  <Button variant="text" onClick={() => complete.mutate({ id: e.id, done: false })}>
                    Not done yet
                  </Button>
                ) : null}
                <Button variant="text" className="text-red" onClick={() => setConfirming(true)}>
                  Delete this event
                </Button>
              </BottomActions>
              <KeepAsMemorySheet
                event={e}
                open={keeping}
                onClose={() => setKeeping(false)}
                onKept={() => router.push(routes.memories)}
              />
              {/* It goes for both of them, so it asks first. There is no undo:
                an event is small enough that re-adding it beats keeping a bin. */}
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
                      remove.mutate(e.id, { onSuccess: () => router.replace(routes.events) })
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
