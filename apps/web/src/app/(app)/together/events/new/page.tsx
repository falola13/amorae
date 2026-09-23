"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { Controller, useForm } from "react-hook-form";

import { BareInput, BareTextarea, ComposeBar, Skeleton } from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { PickRow } from "@/components/ui/pick-row";
import { useEvent, useSaveEvent } from "@/features/together/hooks";
import { eventSchema, type EventInput } from "@/lib/api/schemas";
import type { Event } from "@/lib/api/types";
import { iso, longDate, time12 } from "@/lib/dates";
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
  const existing = useEvent(editId ?? "");

  if (!editId) return <EventComposer />;
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

function EventComposer({ editId, event }: { editId?: string; event?: Event }) {
  const router = useRouter();
  const save = useSaveEvent();
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
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
        }
      : {
          title: "",
          date: iso(today()),
          start_time: "",
          end_time: "",
          location: "",
          reminder: "1 hour before",
          notes: "",
        },
  });

  const onSubmit = (v: EventInput) => {
    const input: EventInput = {
      ...v,
      start_time: v.start_time || undefined,
      end_time: v.end_time || undefined,
      location: v.location || undefined,
      reminder: v.reminder || undefined,
      notes: v.notes || undefined,
    };
    save.mutate(
      { id: editId, input },
      { onSuccess: () => router.replace(editId ? routes.event(editId) : routes.events) },
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
          label="What would you like to do together?"
          placeholder="Date night"
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
                empty={longDate(field.value)}
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
                empty={field.value ? time12(field.value) : "Add a time"}
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
                empty={field.value ? time12(field.value) : "Add a time"}
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
                type="text"
                onChange={field.onChange}
                placeholder="None"
                last
              />
            )}
          />
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
