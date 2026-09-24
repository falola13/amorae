"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";

import { useAddMilestone } from "@/features/together/hooks";
import { milestoneSchema } from "@/lib/api/schemas";
import { iso } from "@/lib/dates";
import { today } from "@/lib/today";
import { BareInput, Button, Sheet, SwitchRow } from "@/components/ui/kit";
import { PickRow } from "@/components/ui/pick-row";

// milestoneSchema covers title/date/sub; whether it repeats every year is a
// plain toggle in this sheet, not something the API schema validates.
const milestoneFormSchema = milestoneSchema.extend({ reminder: z.boolean() });
type MilestoneFormInput = z.infer<typeof milestoneFormSchema>;

export function MilestoneComposer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const add = useAddMilestone();
  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors },
  } = useForm<MilestoneFormInput>({
    resolver: zodResolver(milestoneFormSchema),
    defaultValues: { title: "", date: iso(today()), sub: "", reminder: true },
  });

  const close = () => {
    reset();
    onClose();
  };
  const onSubmit = (v: MilestoneFormInput) =>
    add.mutate(
      { title: v.title, date: v.date, sub: v.sub || undefined, reminder: v.reminder },
      { onSuccess: close },
    );

  return (
    <Sheet open={open} onClose={close} title="A date worth keeping." labelledBy="d-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <BareInput
          label="What is it"
          autoFocus
          placeholder="Our engagement"
          className="h-11 text-[22px] font-semibold tracking-[-0.02em]"
          error={errors.title?.message}
          {...register("title")}
        />
        <Controller
          name="date"
          control={control}
          render={({ field }) => (
            <div className="flex flex-col border-t border-line">
              <PickRow
                icon="calendar"
                label="Date"
                value={field.value}
                type="date"
                onChange={field.onChange}
                last
              />
            </div>
          )}
        />
        <BareInput
          label="A note, optional"
          placeholder="Three years together"
          className="h-10 text-body"
          error={errors.sub?.message}
          {...register("sub")}
        />
        <Controller
          name="reminder"
          control={control}
          render={({ field }) => (
            <SwitchRow
              label="Remind us every year"
              checked={field.value}
              onChange={field.onChange}
              last
            />
          )}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" loading={add.isPending}>
            Keep this date
          </Button>
          <Button type="button" variant="text" onClick={close}>
            Not now
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
