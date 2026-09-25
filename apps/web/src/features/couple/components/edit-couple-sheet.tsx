"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { Alert, Button, Field, Select, Sheet } from "@/components/ui/kit";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { coupleSchema, type CoupleInput } from "@/lib/api/schemas";
import type { Couple } from "@/lib/api/types";
import { notify } from "@/lib/store/toast";
import { ZONES, zoneLabel } from "@/lib/timezones";
import { useUpdateCouple, useUpdateRole } from "../hooks";

// Name/date and role are separate endpoints; only what changed is sent.
function zoneOptions(current?: string) {
  const zones: string[] = [...ZONES];
  if (current && !zones.includes(current)) zones.unshift(current);
  return zones.map((z) => ({ value: z, label: zoneLabel(z) }));
}

export function EditCoupleSheet({
  open,
  onClose,
  couple,
}: {
  open: boolean;
  onClose: () => void;
  couple: Couple;
}) {
  const updateCouple = useUpdateCouple();
  const updateRole = useUpdateRole();
  const initial: CoupleInput = {
    name: couple.name,
    relationship_start_date: couple.started_on ?? "",
    role: couple.me.role ?? "",
    timezone: couple.timezone,
  };
  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors, dirtyFields },
  } = useForm<CoupleInput>({ resolver: zodResolver(coupleSchema), defaultValues: initial });

  useEffect(() => {
    if (open) reset(initial);
    // Re-seed each time the sheet opens, from the couple as it is then.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const close = () => {
    reset(initial);
    onClose();
  };

  const showError = (e: unknown) => {
    const fields = isApiError(e) ? e.fields : undefined;
    if (fields)
      for (const [k, m] of Object.entries(fields)) setError(k as keyof CoupleInput, { message: m });
    else setError("root", { message: isApiError(e) ? e.message : GENERIC_ERROR_MESSAGE });
  };

  const onSubmit = async (v: CoupleInput) => {
    try {
      if (dirtyFields.name || dirtyFields.relationship_start_date || dirtyFields.timezone)
        await updateCouple.mutateAsync({
          name: v.name,
          relationship_start_date: v.relationship_start_date || undefined,
          timezone: dirtyFields.timezone ? v.timezone : undefined,
        });
      if (dirtyFields.role && v.role) await updateRole.mutateAsync(v.role);
      notify("Saved.");
      onClose();
    } catch (e) {
      showError(e);
    }
  };

  return (
    <Sheet open={open} onClose={close} title="Your space" labelledBy="edit-couple-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        {errors.root?.message ? <Alert message={errors.root.message} /> : null}
        <Field label="Name of your space" error={errors.name?.message} {...register("name")} />
        <Field
          label="Together since"
          type="date"
          error={errors.relationship_start_date?.message}
          {...register("relationship_start_date")}
        />
        <Select
          label="Where your week starts"
          hint="Your prayer week turns over on Sunday here, for both of you."
          // Keep the couple's own zone in the list, or saving anything else here would quietly move their week.
          options={zoneOptions(couple.timezone)}
          error={errors.timezone?.message}
          {...register("timezone")}
        />
        <Field
          label="What you call yourself"
          placeholder="Husband, Wife, Partner…"
          hint="Only a label. It changes nothing about what you can do."
          error={errors.role?.message}
          {...register("role")}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" loading={updateCouple.isPending || updateRole.isPending}>
            Save
          </Button>
          <Button type="button" variant="text" onClick={close}>
            Not now
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
