"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useMemo, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { lookupScripture, type LookupFailure, type Scripture } from "@/features/prayers/scripture";
import { parseReference } from "@/lib/scripture/parse";
import type { z } from "zod";

import {
  BareInput,
  BareTextarea,
  Button,
  ComposeBar,
  Para,
  Sheet,
  Skeleton,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { useCouple } from "@/features/couple/hooks";
import { WeekdayChips } from "@/features/prayers/components/weekday-chips";
import { useSavePoints, useWeek } from "@/features/prayers/hooks";
import { prayerPointSchema } from "@/lib/api/schemas";
import { clearDraft, readDraft, writeDraft } from "@/lib/drafts";
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

function EditPrayer() {
  const params = useSearchParams();
  const id = params.get("id");
  // Only meaningful for a brand-new prayer — the week planner's "Add for {day}" links land here
  // with the day preselected; editing an existing point always keeps its own weekdays.
  const dayParam = params.get("day");
  const preselectDay = dayParam !== null && /^[0-6]$/.test(dayParam) ? Number(dayParam) : undefined;
  const week = useWeek();
  const couple = useCouple();
  const partner = couple.data?.partner?.display_name ?? "Your partner";
  const save = useSavePoints();
  const router = useRouter();
  // keepDirtyValues prevents a background refetch (e.g. on window focus) from overwriting unsaved edits.
  const point = week.data?.points.find((x) => x.id === id);
  const {
    register,
    handleSubmit,
    setValue,
    getValues,
    reset,
    watch,
    control,
    formState: { errors, isDirty },
  } = useForm<Form>({
    resolver: zodResolver(prayerPointSchema),
    defaultValues: {
      title: "",
      text: "",
      scripture: "",
      weekdays: preselectDay !== undefined ? [preselectDay] : [],
    },
    values: point
      ? {
          title: point.title,
          text: point.text,
          scripture: point.scripture ?? "",
          weekdays: point.weekdays,
        }
      : undefined,
    resetOptions: { keepDirtyValues: true },
  });

  // The verse text is not a form field. Nobody types it — it arrives from the
  // reference — so it lives here and rides along on save.
  // undefined means "untouched, use whatever the point already has"; null
  // means the person removed it. Avoids an effect just to copy a prop in.
  const [verse, setVerse] = useState<Scripture | null | undefined>(undefined);
  const [looking, setLooking] = useState(false);
  const [failure, setFailure] = useState<LookupFailure | null>(null);
  // One tap on the trash used to delete the prayer outright, notes and verse with it.
  const [confirming, setConfirming] = useState(false);

  // What's typed is kept on the phone as it's typed, and offered back on the
  // way in: a locked phone or a reclaimed tab used to take the draft with it.
  const draftKey = `prayer:${id ?? "new"}`;
  const [restored, setRestored] = useState(false);
  const checked = useRef(false);
  useEffect(() => {
    const sub = watch((v, { type }) => {
      if (type === "change") writeDraft(draftKey, v);
    });
    return () => sub.unsubscribe();
  }, [watch, draftKey]);
  useEffect(() => {
    // Once the point (if any) has loaded, so the draft lands on top of it.
    if (checked.current || !week.isSuccess) return;
    checked.current = true;
    const draft = readDraft<Form>(draftKey);
    if (!draft) return;
    const current = getValues();
    const textKeys = ["title", "text", "scripture"] as const;
    const sameText = textKeys.every((k) => (draft[k] ?? "") === (current[k] ?? ""));
    const sameDays =
      JSON.stringify(draft.weekdays ?? []) === JSON.stringify(current.weekdays ?? []);
    if (sameText && sameDays) return;
    for (const k of textKeys) setValue(k, draft[k] ?? "", { shouldDirty: true });
    setValue("weekdays", draft.weekdays ?? [], { shouldDirty: true });
    setRestored(true);
  }, [week.isSuccess, draftKey, getValues, setValue]);
  const leave = () => {
    clearDraft(draftKey);
    router.replace(routes.prayersSet);
  };
  const startOver = () => {
    clearDraft(draftKey);
    reset(
      point
        ? {
            title: point.title,
            text: point.text,
            scripture: point.scripture ?? "",
            weekdays: point.weekdays,
          }
        : { title: "", text: "", scripture: "", weekdays: [] },
    );
    setRestored(false);
  };

  const saved = point?.verse
    ? { reference: point.scripture ?? "", text: point.verse, translation: "" }
    : null;
  const shown = verse !== undefined ? verse : saved;

  // Parsed locally, with no round trip, so the field can say what it heard as they type.
  // Undefined typed text (nothing yet) and unparseable junk both mean "no preview" — the gentle
  // hint below only shows once they've actually typed something that didn't parse.
  const scriptureInput = watch("scripture");
  const preview = useMemo(
    () => (scriptureInput?.trim() ? parseReference(scriptureInput) : null),
    [scriptureInput],
  );

  const look = async (ref: string) => {
    const trimmed = ref.trim();
    if (!trimmed) {
      setVerse(null);
      setFailure(null);
      return;
    }
    setLooking(true);
    setFailure(null);
    // The parsed, tidied form is what actually gets looked up — "romans 8 28" and "Rom. 8:28"
    // both resolve to the same canonical reference the server expects.
    const result = await lookupScripture(parseReference(trimmed)?.canonical ?? trimmed);
    setLooking(false);
    if (!result.ok) {
      setFailure(result.failure);
      return;
    }
    setVerse(result.scripture);
    // The tidied reference, so "romans 8 28" settles as "Romans 8:28".
    setValue("scripture", result.scripture.reference, { shouldDirty: true });
  };

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
        // Any day this week, not just today — deleting still goes for both of you once either
        // side of the couple has prayed it once.
        const partnerPrayed = id ? w.days.some((d) => d.partner.includes(id)) : false;

        const onSubmit = async (v: Form) => {
          // The lookup runs on blur, and tapping Done straight from the field
          // skips it — the reference saved with no words. Look it up here too;
          // a failure still saves the reference, as it always has.
          let scripture = v.scripture?.trim() || undefined;
          let words = shown?.text;
          // verse === null: they removed the words on purpose; don't bring them back.
          if (scripture && verse !== null && scripture !== shown?.reference) {
            const canonical = parseReference(scripture)?.canonical ?? scripture;
            const found = await lookupScripture(canonical);
            words = found.ok ? found.scripture.text : undefined;
            if (found.ok) scripture = found.scripture.reference;
          }
          const p: PrayerPoint = {
            id: id ?? "",
            title: v.title,
            text: v.text,
            scripture,
            verse: (scripture && words) || undefined,
            position: idx === -1 ? points.length : idx,
            weekdays: v.weekdays ?? [],
          };
          const next =
            idx === -1 ? [...points, p] : points.map((x) => (x.id === id ? { ...x, ...p } : x));
          save.mutate(next, { onSuccess: leave, onQueued: leave });
        };
        const remove = () =>
          save.mutate(
            points.filter((x) => x.id !== id),
            { onSuccess: leave, onQueued: leave },
          );
        const done = () => {
          if (!isDirty && idx !== -1) {
            leave();
            return;
          }
          void handleSubmit(onSubmit)();
        };

        return (
          <>
            <ComposeBar
              onCancel={leave}
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
              {restored ? (
                <p role="status" className="m-0 text-support text-stone">
                  Picked up where you left off.{" "}
                  <button
                    type="button"
                    onClick={startOver}
                    className="press font-semibold text-plum underline"
                  >
                    Start over
                  </button>
                </p>
              ) : null}
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
              <div>
                <p className="m-0 pb-2 text-[13px] font-semibold text-stone">Which days?</p>
                <Controller
                  name="weekdays"
                  control={control}
                  render={({ field }) => (
                    <WeekdayChips value={field.value ?? []} onChange={field.onChange} />
                  )}
                />
              </div>
              <div className="border-t border-line pt-[18px]">
                {/* The reference is all anybody types. The words come from it,
                    because a verse field nobody filled is why scripture has
                    only ever shown as a bare reference. */}
                <BareInput
                  label="Scripture, optional"
                  placeholder="Book, chapter and verse"
                  className="h-11 text-bodylg font-semibold text-plum"
                  {...register("scripture", {
                    onBlur: (e) => {
                      const next = e.target.value.trim();
                      const canonical = parseReference(next)?.canonical;
                      if (next && (canonical ?? next) !== shown?.reference) void look(next);
                      if (!next) setVerse(null);
                    },
                  })}
                />
                {scriptureInput?.trim() && !looking && !failure ? (
                  <p className="m-0 pt-1 text-support text-stone">
                    {preview ? preview.canonical : 'Try Book chapter:verse, like "John 3:16".'}
                  </p>
                ) : null}
                {looking ? (
                  <p className="m-0 pt-1 text-support text-stone">Looking it up&hellip;</p>
                ) : null}
                {failure ? (
                  // Never an error state on the field: the reference is still
                  // theirs, it still saves, and the words can be added later.
                  <p className="m-0 pt-1 text-support text-stone">
                    {failure.message}{" "}
                    <button
                      type="button"
                      onClick={() => void look(getValues("scripture") ?? "")}
                      className="press font-semibold text-plum underline"
                    >
                      Try again
                    </button>
                  </p>
                ) : null}
                {shown?.text && !looking ? (
                  <div className="flex flex-col gap-1 pt-2.5">
                    <p className="m-0 text-[15px] leading-[1.55] text-stone" data-selectable>
                      {shown.text}
                    </p>
                    <div className="flex items-center gap-3">
                      {shown.translation ? (
                        <span className="text-[13px] text-edge">{shown.translation}</span>
                      ) : null}
                      <button
                        type="button"
                        onClick={() => {
                          setVerse(null);
                          setFailure(null);
                        }}
                        className="press text-[13px] font-semibold text-plum"
                      >
                        Remove the words
                      </button>
                    </div>
                  </div>
                ) : null}
              </div>
              {idx !== -1 ? (
                <div className="border-t border-line pt-1.5">
                  <Button
                    variant="danger"
                    icon="trash"
                    onClick={() => setConfirming(true)}
                    className="w-auto justify-start px-0"
                  >
                    Delete this prayer
                  </Button>
                </div>
              ) : null}
              <Sheet
                open={confirming}
                onClose={() => setConfirming(false)}
                title="Delete this prayer?"
                labelledBy="del-prayer-h"
              >
                <Para>
                  {partnerPrayed
                    ? `${partner} has prayed this one — it goes for both of you.`
                    : `“${points[idx]?.title}” goes for both of you, with its words and scripture. There is no undo.`}
                </Para>
                <div className="flex flex-col gap-1">
                  <Button
                    variant="secondary"
                    className="border-red text-red"
                    loading={save.isPending}
                    onClick={remove}
                  >
                    Delete it
                  </Button>
                  <Button variant="text" onClick={() => setConfirming(false)}>
                    Keep it
                  </Button>
                </div>
              </Sheet>
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
