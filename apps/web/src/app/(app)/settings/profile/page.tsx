"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";

import { Main } from "@/components/layout/screen";
import {
  BottomActions,
  Button,
  Field,
  Initial,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { ChangeEmailSheet } from "@/features/couple/components/change-email-sheet";
import { ChangePasswordSheet } from "@/features/couple/components/change-password-sheet";
import { EditCoupleSheet } from "@/features/couple/components/edit-couple-sheet";
import { useCouple, useUpdateProfile } from "@/features/couple/hooks";
import { isApiError } from "@/lib/api/errors";
import { profileSchema, type ProfileInput } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";

const ZONES = ["Africa/Lagos", "America/New_York", "Europe/London", "Africa/Nairobi", "Asia/Dubai"];

export default function ProfilePage() {
  const couple = useCouple();
  const me = couple.data?.me;
  const update = useUpdateProfile();
  const [saved, setSaved] = useState(false);
  const [changingEmail, setChangingEmail] = useState(false);
  const [changingPassword, setChangingPassword] = useState(false);
  const [editingCouple, setEditingCouple] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    setError,
    control,
    formState: { errors, isDirty },
  } = useForm<ProfileInput>({
    resolver: zodResolver(profileSchema),
    defaultValues: { display_name: "", timezone: "Africa/Lagos" },
    // Straight from the profile, rather than an effect calling reset() on
    // every couple.data change: that also fires on a background refetch and
    // would wipe a half-typed name. keepDirtyValues protects edited fields.
    values: me ? { display_name: me.display_name, timezone: me.timezone } : undefined,
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
        queries={[couple]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(coupleData) => {
          const partner = coupleData.partner?.display_name;
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
                  <button
                    type="button"
                    onClick={() => setChangingEmail(true)}
                    className="press flex h-[54px] items-center justify-between gap-3 border-y border-line text-left text-[16px] font-medium text-ink"
                  >
                    <span className="flex min-w-0 flex-col">
                      <span className="text-[13px] font-semibold text-stone">Email</span>
                      <span className="truncate">{coupleData.me.email}</span>
                    </span>
                    <span className="shrink-0 text-[15px] font-semibold text-plum">Change</span>
                  </button>
                  <div className="flex flex-col gap-2">
                    <label htmlFor="timezone" className="text-[13px] font-semibold text-stone">
                      Timezone
                    </label>
                    <select
                      id="timezone"
                      className="h-[52px] w-full appearance-none rounded-input border border-edge bg-surface px-4 text-[16px] text-ink"
                      {...register("timezone")}
                    >
                      {ZONES.map((z) => (
                        <option key={z} value={z}>
                          {z.replace("_", " ")}
                        </option>
                      ))}
                    </select>
                  </div>
                  <button
                    type="button"
                    onClick={() => setChangingPassword(true)}
                    className="press flex h-[54px] items-center justify-between border-y border-line text-left text-[16px] font-medium text-ink"
                  >
                    Password
                    <span className="text-[15px] font-semibold text-plum">Change</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditingCouple(true)}
                    className="press flex h-[54px] items-center justify-between gap-3 border-b border-line text-left text-[16px] font-medium text-ink"
                  >
                    <span className="flex min-w-0 flex-col">
                      <span className="text-[13px] font-semibold text-stone">Your space</span>
                      <span className="truncate">
                        {coupleData.name}
                        {coupleData.me.role ? ` · you’re ${coupleData.me.role}` : ""}
                      </span>
                    </span>
                    <span className="shrink-0 text-[15px] font-semibold text-plum">Edit</span>
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
      {couple.data ? (
        <EditCoupleSheet
          open={editingCouple}
          onClose={() => setEditingCouple(false)}
          couple={couple.data}
        />
      ) : null}
    </>
  );
}
