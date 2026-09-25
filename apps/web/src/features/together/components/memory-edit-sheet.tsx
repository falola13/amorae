"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { useUpdateMemory } from "@/features/together/hooks";
import { memorySchema } from "@/lib/api/schemas";
import type { Memory } from "@/lib/api/types";
import { BareInput, BareTextarea, Button, Field, Sheet } from "@/components/ui/kit";

// The composer files a moment under today; editing is the one place its day can move.
const schema = memorySchema.extend({
  date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Pick a date."),
});
type Input = z.infer<typeof schema>;

/** The words and the day of a kept moment. The photo keeps its own actions. */
export function MemoryEditSheet({ memory, onClose }: { memory: Memory; onClose: () => void }) {
  const update = useUpdateMemory();
  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
  } = useForm<Input>({
    resolver: zodResolver(schema),
    defaultValues: {
      title: memory.title,
      date: memory.date.slice(0, 10),
      location: memory.location ?? "",
      note: memory.note ?? "",
    },
  });

  const onSubmit = (v: Input) =>
    update.mutate(
      {
        id: memory.id,
        title: v.title,
        date: v.date,
        location: v.location || undefined,
        note: v.note || undefined,
      },
      { onSuccess: onClose, onQueued: onClose },
    );

  return (
    <Sheet open onClose={onClose} dirty={isDirty} title="This moment" labelledBy="mem-edit-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <BareInput
          label="What happened"
          className="h-11 text-[22px] font-semibold tracking-[-0.02em]"
          error={errors.title?.message}
          {...register("title")}
        />
        <Field label="When" type="date" error={errors.date?.message} {...register("date")} />
        <BareInput
          label="Where, optional"
          className="h-10 text-body"
          error={errors.location?.message}
          {...register("location")}
        />
        <BareTextarea
          label="A short note, optional"
          rows={2}
          className="min-h-[56px] text-body"
          error={errors.note?.message}
          {...register("note")}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" loading={update.isPending}>
            Save changes
          </Button>
          <Button type="button" variant="text" onClick={onClose}>
            Not now
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
