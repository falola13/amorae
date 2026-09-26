"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";

import { Main } from "@/components/layout/screen";
import {
  BottomActions,
  Button,
  Field,
  Initial,
  Select,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { ChangeEmailSheet } from "@/features/couple/components/change-email-sheet";
import { ChangePasswordSheet } from "@/features/couple/components/change-password-sheet";
import { useCouple, useMe, useUpdateProfile } from "@/features/couple/hooks";
import { BirthdayField } from "@/features/settings/components/birthday-field";
import { isApiError } from "@/lib/api/errors";
import { profileSchema, type ProfileInput } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";
import { ZONES, zoneLabel } from "@/lib/timezones";

export default function ProfilePage() {
  // Reads from the account, not the couple, so it works without a paired partner.
  const account = useMe();
  const me = account.data;
  const couple = useCouple();
  const update = useUpdateProfile();
  const [saved, setSaved] = useState(false);
  const [changingEmail, setChangingEmail] = useState(false);
  const [changingPassword, setChangingPassword] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    setError,
    control,
    formState: { errors, isDirty },
  } = useForm<ProfileInput>({
    resolver: zodResolver(profileSchema),
    defaultValues: { display_name: "", timezone: "Africa/Lagos", birthday: null },
    // keepDirtyValues prevents a background refetch from wiping a half-typed name.
    values: me
      ? { display_name: me.display_name, timezone: me.timezone, birthday: me.birthday ?? null }
      : undefined,
    resetOptions: { keepDirtyValues: true },
  });
  const name = useWatch({ control, name: "display_name" });
  const onSubmit = (v: ProfileInput) =>
    update.mutate(v, {
      onSuccess: () => {
        reset(v);
        setSaved(true);
        setTimeout(() => setSaved(false), 2000);
      },
      onError: (e) => {
        if (isApiError(e))
          for (const [k, m] of Object.entries(e.fields ?? {}))
            setError(k as keyof ProfileInput, { message: m });
      },
    });
  return (
    <>
      <TopBar back="Settings" backHref={routes.settings} />
      <QueryState
        queries={[account]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(profile) => {
          const partner = couple.data?.partner?.display_name;
          return (
            <form
              method="post"
              onSubmit={handleSubmit(onSubmit)}
              className="flex grow flex-col"
              noValidate
            >
              <Main className="gap-[26px] pt-3">
                <Title>Profile</Title>
                <div className="flex items-center gap-4">
                  <Initial letter={name?.[0] ?? "A"} size={64} />
                  <div className="flex flex-col">
                    <div className="text-bodylg font-semibold">{name || "You"}</div>
                    <div className="text-support text-stone">
                      {partner ? `Praying with ${partner}` : "Waiting for your partner"}
                    </div>
                  </div>
                </div>
                <div className="flex flex-col gap-[18px]">
                  <Field
                    label="First name"
                    autoComplete="given-name"
                    hint={
                      partner
                        ? `This is what ${partner} sees.`
                        : "This is what your partner will see."
                    }
                    error={errors.display_name?.message}
                    {...register("display_name")}
                  />
                  <Controller
                    name="birthday"
                    control={control}
                    render={({ field }) => (
                      <BirthdayField
                        value={field.value}
                        onChange={field.onChange}
                        hint={`${partner ?? "Your partner"} gets a reminder before it — you won’t get one for your own.`}
                        error={errors.birthday?.message}
                      />
                    )}
                  />
                  <button
                    type="button"
                    onClick={() => setChangingEmail(true)}
                    className="press flex h-[54px] items-center justify-between gap-3 border-y border-line text-left text-[16px] font-medium text-ink"
                  >
                    <span className="flex min-w-0 flex-col">
                      <span className="text-[13px] font-semibold text-stone">Email</span>
                      <span className="truncate">{profile.email}</span>
                    </span>
                    <span className="shrink-0 text-[15px] font-semibold text-plum">Change</span>
                  </button>
                  <Select
                    label="Timezone"
                    hint="When your own reminders arrive. Your prayer week follows your space's timezone."
                    options={ZONES.map((z) => ({ value: z, label: zoneLabel(z) }))}
                    {...register("timezone")}
                  />
                  <button
                    type="button"
                    onClick={() => setChangingPassword(true)}
                    className="press flex h-[54px] items-center justify-between border-y border-line text-left text-[16px] font-medium text-ink"
                  >
                    Password
                    <span className="text-[15px] font-semibold text-plum">Change</span>
                  </button>
                </div>
              </Main>
              <BottomActions>
                <Button
                  type="submit"
                  disabled={!isDirty && !saved}
                  loading={update.isPending}
                  icon={saved ? "check" : undefined}
                >
                  {saved ? "Saved" : "Save changes"}
                </Button>
              </BottomActions>
            </form>
          );
        }}
      </QueryState>
      <ChangeEmailSheet open={changingEmail} onClose={() => setChangingEmail(false)} />
      <ChangePasswordSheet open={changingPassword} onClose={() => setChangingPassword(false)} />
    </>
  );
}
