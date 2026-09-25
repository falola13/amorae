"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";

import { BareInput, BareTextarea, ComposeBar, Para, Skeleton, cx } from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { PickRow } from "@/components/ui/pick-row";
import { useCouple } from "@/features/couple/hooks";
import { canChangeEventKind, REMINDER_OPTIONS } from "@/features/together/events";
import { useEvent, useSaveEvent } from "@/features/together/hooks";
import { usePrefs } from "@/features/settings/hooks";
import { eventSchema, type EventInput } from "@/lib/api/schemas";
import type { Event } from "@/lib/api/types";
import { iso } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";

export default function NewEventPage() {
  return (
    <Suspense fallback={null}>
      <EventForm />
    </Suspense>
  );
}

function EventForm() {
  const params = useSearchParams();
  const editId = params.get("edit") ?? undefined;
  // Validated as a date since it comes from a URL param.
  const on = params.get("on");
  const startOn = on && /^\d{4}-\d{2}-\d{2}$/.test(on) ? on : undefined;
  const existing = useEvent(editId ?? "");

  if (!editId) return <EventComposer startOn={startOn} />;
  return (
    <QueryState
      queries={[existing]}
      frame={inPage}
      loading={
        <>
          <ComposeBar cancelHref={routes.events} label="Edit event" busy />
          <Skeleton />
        </>
      }
    >
      {(event) => <EventComposer editId={editId} event={event} />}
    </QueryState>
  );
}

function EventComposer({
  editId,
  event,
  startOn,
}: {
  editId?: string;
  event?: Event;
  startOn?: string;
}) {
  const router = useRouter();
  const save = useSaveEvent();
  const couple = useCouple();
  const prefs = usePrefs();
  const meId = couple.data?.me.id;
  const partnerName = couple.data?.partner?.display_name;
  // Ticking checklist items off happens on the event page, not here.
  const [checklist, setChecklist] = useState(
    event?.checklist.map((item) => item.text).join("\n") ?? "",
  );
  const {
    register,
    handleSubmit,
    control,
    watch,
    setValue,
    formState: { errors, dirtyFields },
  } = useForm<EventInput>({
    resolver: zodResolver(eventSchema),
    defaultValues: event
      ? {
          title: event.title,
          date: event.date,
          start_time: event.start_time ?? "",
          end_time: event.end_time ?? "",
          location: event.location ?? "",
          reminder: event.reminder ?? "",
          notes: event.notes ?? "",
          kind: event.kind,
        }
      : {
          title: "",
          date: startOn ?? iso(today()),
          start_time: "",
          end_time: "",
          location: "",
          // What new events had before there was a setting for it.
          reminder: "1 hour before",
          notes: "",
          kind: "together",
        },
  });

  // New events only: once the person's default shows up, use it — but not
  // over a reminder they've already touched, and never on an edit (that one
  // keeps whatever the event already had).
  useEffect(() => {
    // Checked against undefined, not falsy: "" is a real choice (no reminder).
    if (event || dirtyFields.reminder || prefs.data?.default_event_reminder === undefined) return;
    setValue("reminder", prefs.data.default_event_reminder);
  }, [event, prefs.data?.default_event_reminder, dirtyFields.reminder, setValue]);

  const kind = watch("kind");
  // Only the creator gets to change this; an edit by anyone else keeps it hidden.
  const canPickKind = !event || canChangeEventKind(event, meId);

  const onSubmit = (v: EventInput) => {
    const input: EventInput = {
      ...v,
      checklist: checklist.split("\n"),
      start_time: v.start_time || undefined,
      end_time: v.end_time || undefined,
      location: v.location || undefined,
      reminder: v.reminder || undefined,
      notes: v.notes || undefined,
      kind: canPickKind ? v.kind : undefined,
    };
    save.mutate(
      { id: editId, input },
      {
        onSuccess: () => router.replace(editId ? routes.event(editId) : routes.events),
        onQueued: () => router.replace(editId ? routes.event(editId) : routes.events),
      },
    );
  };

  return (
    <>
      <ComposeBar
        cancelHref={editId ? routes.event(editId) : routes.events}
        label={editId ? "Edit event" : "New event"}
        onDone={() => void handleSubmit(onSubmit)()}
        busy={save.isPending}
      />
      <form
        method="post"
        onSubmit={handleSubmit(onSubmit)}
        className="flex grow flex-col px-6 pt-5"
        noValidate
      >
        <BareInput
          // "together" in the question would be wrong for a just-me plan.
          label={kind === "mine" ? "What’s the plan?" : "What would you like to do together?"}
          placeholder={kind === "mine" ? "Dentist, a work dinner…" : "Date night"}
          autoFocus={!editId}
          className="h-11 text-title"
          aria-invalid={!!errors.title}
          {...register("title")}
        />
        {errors.title ? (
          <div role="alert" className="mt-1 text-[13px] text-red">
            {errors.title.message}
          </div>
        ) : null}
        {canPickKind ? (
          <Controller
            name="kind"
            control={control}
            render={({ field }) => (
              <div className="mt-4 flex flex-col gap-1.5">
                <div className="text-[13px] font-semibold text-stone">Who&rsquo;s it for?</div>
                <div role="radiogroup" aria-label="Who's it for?" className="-mx-1 flex gap-1">
                  {(
                    [
                      { value: "together", label: "Both of us" },
                      { value: "mine", label: "Just me" },
                    ] as const
                  ).map((o) => (
                    <button
                      key={o.value}
                      type="button"
                      role="radio"
                      aria-checked={field.value === o.value}
                      onClick={() => field.onChange(o.value)}
                      className={cx(
                        "press h-9 rounded-full px-3.5 text-[14px] font-semibold",
                        field.value === o.value ? "bg-plum-tint text-plum" : "text-stone",
                      )}
                    >
                      {o.label}
                    </button>
                  ))}
                </div>
              </div>
            )}
          />
        ) : null}
        <div className="mt-[18px] flex flex-col border-t border-line">
          <Controller
            name="date"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="calendar"
                label="Date"
                value={field.value}
                type="date"
                onChange={field.onChange}
              />
            )}
          />
          <Controller
            name="start_time"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="clock"
                label="Starts"
                value={field.value ?? ""}
                type="time"
                onChange={field.onChange}
                placeholder="Add a time"
              />
            )}
          />
          <Controller
            name="end_time"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="clock"
                label="Ends"
                value={field.value ?? ""}
                type="time"
                onChange={field.onChange}
                placeholder="Add a time"
              />
            )}
          />
          <Controller
            name="location"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="pin"
                label="Location"
                value={field.value ?? ""}
                type="text"
                onChange={field.onChange}
                placeholder="Add a place"
              />
            )}
          />
          <Controller
            name="reminder"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="bell"
                label="Reminder"
                value={field.value ?? ""}
                options={REMINDER_OPTIONS}
                onChange={field.onChange}
                placeholder="None"
                last
              />
            )}
          />
        </div>
        <Para size="support" className="mt-1.5">
          {/* Both of you see every event; what differs is who's reminded. */}
          {kind === "mine"
            ? `${partnerName ?? "Your partner"} can see it, but only you get reminded.`
            : "You’ll both get the reminder."}
        </Para>
        <div className="mt-5">
          <BareTextarea
            label="Checklist"
            rows={3}
            placeholder={"Book the table\nAsk about parking"}
            className="min-h-[84px] text-bodylg"
            value={checklist}
            onChange={(e) => setChecklist(e.target.value)}
          />
          <div className="pt-1.5 text-[13px] text-stone">
            One per line. You can tick these off on the day.
          </div>
        </div>
        <div className="mt-5">
          <BareTextarea
            label="Notes"
            rows={3}
            placeholder="Anything worth remembering for the day."
            className="min-h-[84px] text-bodylg"
            {...register("notes")}
          />
        </div>
        <button type="submit" className="sr-only">
          Save
        </button>
      </form>
    </>
  );
}
