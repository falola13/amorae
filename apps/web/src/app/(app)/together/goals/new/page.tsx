"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { Controller, useForm, useWatch } from "react-hook-form";

import { BareTextarea, ComposeBar } from "@/components/ui/kit";
import { PickRow } from "@/components/ui/pick-row";
import { useCreateGoal } from "@/features/together/hooks";
import { goalSchema, type GoalInput } from "@/lib/api/schemas";
import { addDays, iso, longDate, naira } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";

export default function NewGoalPage() {
  const router = useRouter();
  const create = useCreateGoal();
  const {
    register,
    handleSubmit,
    control,
    formState: { errors },
  } = useForm<GoalInput>({
    resolver: zodResolver(goalSchema),
    defaultValues: {
      title: "",
      why: "",
      target: 0,
      unit: "naira",
      start_date: iso(today()),
      end_date: iso(addDays(today(), 90)),
    },
  });
  const unit = useWatch({ control, name: "unit" });
  const onSubmit = (v: GoalInput) =>
    create.mutate(
      { ...v, why: v.why || undefined },
      { onSuccess: () => router.replace(routes.goals) },
    );
  return (
    <>
      <ComposeBar
        cancelHref={routes.goals}
        label="New goal"
        onDone={() => void handleSubmit(onSubmit)()}
        busy={create.isPending}
      />
      <form
        method="post"
        onSubmit={handleSubmit(onSubmit)}
        className="flex grow flex-col px-6 pt-5"
        noValidate
      >
        <BareTextarea
          label="What would you like to build together?"
          rows={2}
          placeholder="Save &#8358;500,000 together"
          autoFocus
          className="min-h-[72px] text-title leading-[1.25]"
          aria-invalid={!!errors.title}
          {...register("title")}
        />
        {errors.title ? (
          <div role="alert" className="mt-1 text-[13px] text-red">
            {errors.title.message}
          </div>
        ) : null}
        <div className="mt-4">
          <BareTextarea
            label="Why it matters"
            rows={2}
            placeholder="So December feels generous instead of tight."
            className="min-h-[56px] text-bodylg"
            {...register("why")}
          />
        </div>
        <div className="mt-[18px] flex flex-col border-t border-line">
          <Controller
            name="target"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="target"
                label="Target"
                value={field.value ? String(field.value) : ""}
                type="text"
                onChange={(v) => field.onChange(Number(v.replace(/[^\d]/g, "")) || 0)}
                placeholder={unit === "naira" ? "₦500,000" : "A number"}
                empty={
                  field.value
                    ? unit === "naira"
                      ? naira(field.value)
                      : String(field.value)
                    : undefined
                }
              />
            )}
          />
          {errors.target ? (
            <div role="alert" className="py-1 text-[13px] text-red">
              {errors.target.message}
            </div>
          ) : null}
          <Controller
            name="unit"
            control={control}
            render={({ field }) => (
              <div className="flex h-[54px] items-center gap-3.5 border-b border-line text-[16px] font-medium">
                <span className="w-[22px]" />
                <span className="grow">Counted in</span>
                <div role="radiogroup" aria-label="Unit" className="flex gap-1">
                  {(["naira", "count"] as const).map((u) => (
                    <button
                      key={u}
                      type="button"
                      role="radio"
                      aria-checked={field.value === u}
                      onClick={() => field.onChange(u)}
                      className={`press h-9 rounded-full px-3.5 text-[14px] font-semibold ${field.value === u ? "bg-plum-tint text-plum" : "text-stone"}`}
                    >
                      {u === "naira" ? "Naira" : "Steps"}
                    </button>
                  ))}
                </div>
              </div>
            )}
          />
          <Controller
            name="start_date"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="calendar"
                label="Starts"
                value={field.value}
                type="date"
                onChange={field.onChange}
                empty={longDate(field.value)}
              />
            )}
          />
          <Controller
            name="end_date"
            control={control}
            render={({ field }) => (
              <PickRow
                icon="calendar"
                label="Ends"
                value={field.value}
                type="date"
                onChange={field.onChange}
                empty={longDate(field.value)}
                last
              />
            )}
          />
        </div>
        <button type="submit" className="sr-only">
          Save
        </button>
      </form>
    </>
  );
}
