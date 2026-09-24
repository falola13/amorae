"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";
import { useForm } from "react-hook-form";
import type { z } from "zod";

import { Main } from "@/components/layout/screen";
import {
  BareInput,
  BareTextarea,
  Button,
  ComposeBar,
  LinkButton,
  Para,
  Skeleton,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { useCouple } from "@/features/couple/hooks";
import { useSavePoints, useWeek } from "@/features/prayers/hooks";
import { prayerPointSchema } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";
import type { PrayerPoint } from "@/lib/api/types";

type Form = z.infer<typeof prayerPointSchema>;

export default function EditPrayerPage() {
  return (
    <Suspense fallback={null}>
      <EditPrayer />
    </Suspense>
  );
}

/** Feels like writing a note, not filling a form: bare fields, big type. */
function EditPrayer() {
  const params = useSearchParams();
  const id = params.get("id");
  const week = useWeek();
  const couple = useCouple();
  const partner = couple.data?.partner?.display_name ?? "Your partner";
  const save = useSavePoints();
  const router = useRouter();
  // The point's own values, handed to the form directly. An effect calling
  // reset() on every week.data change would also fire on a background
  // refetch — queries refetch when the window regains focus — and overwrite
  // whatever the person had typed. keepDirtyValues leaves edited fields alone.
  const point = week.data?.points.find((x) => x.id === id);
  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
  } = useForm<Form>({
    resolver: zodResolver(prayerPointSchema),
    defaultValues: { title: "", text: "", scripture: "" },
    values: point
      ? { title: point.title, text: point.text, scripture: point.scripture ?? "" }
      : undefined,
    resetOptions: { keepDirtyValues: true },
  });

  return (
    <QueryState
      queries={[week]}
      frame={inPage}
      loading={
        <div className="px-6 pt-16">
          <Skeleton />
        </div>
      }
    >
      {(w) => {
        const points = w.points;
        const idx = points.findIndex((x) => x.id === id);

        // The list hides the link to a prayer the partner has prayed, but a
        // back button or an old tab can still land here. Saying so beats
        // letting them retype it and meet a 409 at the end.
        if (idx !== -1 && id && w.partner_completed.includes(id)) {
          return (
            <Main>
              <div className="pt-3">
                <h1 className="m-0 text-[30px] font-semibold leading-[1.18] tracking-[-0.022em]">
                  {points[idx].title}
                </h1>
                <Para className="mt-1.5">
                  {partner} has already prayed this one, so it stays as it is. You can still add new
                  prayers, or change the ones they haven&rsquo;t reached.
                </Para>
              </div>
              <div className="pt-6">
                <LinkButton href={routes.prayersSet} variant="secondary">
                  Back to the week
                </LinkButton>
              </div>
            </Main>
          );
        }

        const onSubmit = (v: Form) => {
          const p: PrayerPoint = {
            id: id ?? "",
            title: v.title,
            text: v.text,
            scripture: v.scripture || undefined,
            position: idx === -1 ? points.length : idx,
          };
          const next =
            idx === -1 ? [...points, p] : points.map((x) => (x.id === id ? { ...x, ...p } : x));
          save.mutate(next, { onSuccess: () => router.replace(routes.prayersSet) });
        };
        const remove = () =>
          save.mutate(
            points.filter((x) => x.id !== id),
            { onSuccess: () => router.replace(routes.prayersSet) },
          );
        const done = () => {
          if (!isDirty && idx !== -1) {
            router.replace(routes.prayersSet);
            return;
          }
          void handleSubmit(onSubmit)();
        };

        return (
          <>
            <ComposeBar
              cancelHref={routes.prayersSet}
              label={idx === -1 ? "New prayer" : `Prayer ${idx + 1} of ${points.length}`}
              done="Done"
              onDone={done}
              busy={save.isPending}
            />
            <form
              method="post"
              onSubmit={handleSubmit(onSubmit)}
              className="flex grow flex-col gap-[22px] px-6 pt-5"
              noValidate
            >
              <div>
                <BareInput
                  label="Title"
                  placeholder="What are we praying for?"
                  autoFocus={idx === -1}
                  className="h-11 text-title"
                  aria-invalid={!!errors.title}
                  {...register("title")}
                />
                {errors.title ? (
                  <div role="alert" className="mt-1 text-[13px] text-red">
                    {errors.title.message}
                  </div>
                ) : null}
              </div>
              <BareTextarea
                label="Prayer"
                rows={4}
                placeholder="Write it the way you’d say it."
                className="min-h-[124px] text-[19px] leading-[1.6]"
                {...register("text")}
              />
              <div className="border-t border-line pt-[18px]">
                <BareInput
                  label="Scripture, optional"
                  placeholder="Book, chapter and verse"
                  className="h-11 text-bodylg font-semibold text-plum"
                  {...register("scripture")}
                />
              </div>
              {idx !== -1 ? (
                <div className="border-t border-line pt-1.5">
                  <Button
                    variant="danger"
                    icon="trash"
                    onClick={remove}
                    className="w-auto justify-start px-0"
                  >
                    Delete this prayer
                  </Button>
                </div>
              ) : null}
              <button type="submit" className="sr-only">
                Done
              </button>
            </form>
          </>
        );
      }}
    </QueryState>
  );
}
