"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import type { z } from "zod";

import { useAddJournal, useDeleteJournal, useUpdateJournal } from "@/features/together/hooks";
import { journalSchema } from "@/lib/api/schemas";
import type { JournalEntry } from "@/lib/api/types";
import { BareTextarea, Button, Para, Sheet, cx } from "@/components/ui/kit";

const TAGS: JournalEntry["tag"][] = ["Gratitude", "Reflection", "Memory", "Appreciation", "Plans"];

type JournalFormInput = z.infer<typeof journalSchema>;

/** Writes a new entry, or — given one of your own — edits or deletes it. Keyed
 *  by the entry where it's used, so each opening starts from that entry. */
export function JournalComposer({
  open,
  onClose,
  entry,
}: {
  open: boolean;
  onClose: () => void;
  entry?: JournalEntry | null;
}) {
  const add = useAddJournal();
  const update = useUpdateJournal();
  const remove = useDeleteJournal();
  const [deleting, setDeleting] = useState(false);
  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors, isDirty },
  } = useForm<JournalFormInput>({
    resolver: zodResolver(journalSchema),
    defaultValues: entry ? { tag: entry.tag, text: entry.text } : { tag: "Gratitude", text: "" },
  });

  const close = () => {
    reset();
    setDeleting(false);
    onClose();
  };
  const onSubmit = (v: JournalFormInput) =>
    entry
      ? update.mutate({ id: entry.id, ...v }, { onSuccess: close, onQueued: close })
      : add.mutate(v, { onSuccess: close, onQueued: close });

  if (entry && deleting) {
    return (
      <Sheet open={open} onClose={close} title="Delete this entry?" labelledBy="j-del-h">
        <Para>It goes for both of you. There is no undo.</Para>
        <div className="flex flex-col gap-1">
          <Button
            variant="secondary"
            className="border-red text-red"
            loading={remove.isPending}
            onClick={() => remove.mutate(entry.id, { onSuccess: close, onQueued: close })}
          >
            Delete it
          </Button>
          <Button variant="text" onClick={() => setDeleting(false)}>
            Keep it
          </Button>
        </div>
      </Sheet>
    );
  }

  return (
    <Sheet
      open={open}
      onClose={close}
      dirty={isDirty}
      title={entry ? "Your entry" : "Write something for us."}
      labelledBy="j-h"
    >
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <Controller
          name="tag"
          control={control}
          render={({ field }) => (
            <div
              role="radiogroup"
              aria-label="Kind of entry"
              className="-mx-1 flex flex-wrap gap-1"
            >
              {TAGS.map((t) => (
                <button
                  key={t}
                  type="button"
                  role="radio"
                  aria-checked={field.value === t}
                  onClick={() => field.onChange(t)}
                  className={cx(
                    "press h-9 rounded-full px-3.5 text-[14px] font-semibold",
                    field.value === t ? "bg-plum-tint text-plum" : "text-stone",
                  )}
                >
                  {t}
                </button>
              ))}
            </div>
          )}
        />
        <BareTextarea
          label="Entry"
          hideLabel
          rows={4}
          autoFocus
          placeholder="A line or two, in your own words."
          className="min-h-[110px] text-[19px] leading-[1.6]"
          error={errors.text?.message}
          {...register("text")}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" loading={add.isPending || update.isPending}>
            {entry ? "Save changes" : "Save to our journal"}
          </Button>
          <Button type="button" variant="text" onClick={close}>
            Not now
          </Button>
          {entry ? (
            <Button type="button" variant="danger" icon="trash" onClick={() => setDeleting(true)}>
              Delete this entry
            </Button>
          ) : null}
        </div>
      </form>
    </Sheet>
  );
}
