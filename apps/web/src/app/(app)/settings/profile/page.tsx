"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRef, useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";

import { Main } from "@/components/layout/screen";
import {
  BottomActions,
  Button,
  Field,
  Initial,
  Select,
  Skeleton,
  Spinner,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { ChangeEmailSheet } from "@/features/couple/components/change-email-sheet";
import { ChangePasswordSheet } from "@/features/couple/components/change-password-sheet";
import {
  MAX_PHOTO_BYTES,
  useCouple,
  useMe,
  useRemoveMyPhoto,
  useUpdateProfile,
  useUploadMyPhoto,
} from "@/features/couple/hooks";
import { BirthdayField } from "@/features/settings/components/birthday-field";
import { isApiError } from "@/lib/api/errors";
import { profileSchema, type ProfileInput } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";
import { ZONES, zoneLabel } from "@/lib/timezones";

// "photos_unavailable": the server has no photo storage configured. Everything
// else the upload can fail with gets the generic retry message.
function photoErrorMessage(err: unknown): string {
  if (isApiError(err) && err.code === "photos_unavailable") return "Photos aren’t set up yet.";
  return "That didn’t work. Try again.";
}

export default function ProfilePage() {
  // Reads from the account, not the couple, so it works without a paired partner.
  const account = useMe();
  const me = account.data;
  const couple = useCouple();
  const update = useUpdateProfile();
  const uploadPhoto = useUploadMyPhoto();
  const removePhoto = useRemoveMyPhoto();
  const [saved, setSaved] = useState(false);
  const [changingEmail, setChangingEmail] = useState(false);
  const [changingPassword, setChangingPassword] = useState(false);
  const [photoError, setPhotoError] = useState<string | null>(null);
  const photoInput = useRef<HTMLInputElement>(null);

  const choosePhoto = (file: File | null) => {
    if (!file) return;
    setPhotoError(null);
    if (file.size > MAX_PHOTO_BYTES) {
      setPhotoError("That photo is over 10MB. Pick a smaller one.");
      return;
    }
    uploadPhoto.mutate(file, { onError: (e) => setPhotoError(photoErrorMessage(e)) });
  };
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
                  <div className="relative">
                    <Initial letter={name?.[0] ?? "A"} photoUrl={profile.photo_url} size={64} />
                    {uploadPhoto.isPending ? (
                      <span className="absolute inset-0 flex items-center justify-center rounded-full bg-ink/40">
                        <Spinner />
                      </span>
                    ) : null}
                  </div>
                  <div className="flex flex-col gap-1">
                    <div className="text-bodylg font-semibold">{name || "You"}</div>
                    <div className="text-support text-stone">
                      {partner ? `Praying with ${partner}` : "Waiting for your partner"}
                    </div>
                    <div className="flex items-center gap-2.5 pt-0.5">
                      <button
                        type="button"
                        onClick={() => photoInput.current?.click()}
                        disabled={uploadPhoto.isPending}
                        className="press text-[14px] font-semibold text-plum"
                      >
                        {profile.photo_url ? "Change photo" : "Add a photo"}
                      </button>
                      {profile.photo_url ? (
                        <button
                          type="button"
                          onClick={() =>
                            removePhoto.mutate(undefined, {
                              onError: (e) => setPhotoError(photoErrorMessage(e)),
                            })
                          }
                          disabled={removePhoto.isPending}
                          className="press text-[14px] font-semibold text-stone"
                        >
                          Remove
                        </button>
                      ) : null}
                    </div>
                  </div>
                  <input
                    ref={photoInput}
                    type="file"
                    accept="image/*"
                    className="sr-only"
                    onChange={(e) => {
                      choosePhoto(e.target.files?.[0] ?? null);
                      e.target.value = "";
                    }}
                  />
                </div>
                {photoError ? (
                  <div role="alert" className="text-[13px] text-red">
                    {photoError}
                  </div>
                ) : null}
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
                        // A bad day or year is reported on the nested field, which the
                        // top-level message alone never showed — a silent failed save.
                        error={
                          errors.birthday?.message ??
                          errors.birthday?.day?.message ??
                          errors.birthday?.year?.message ??
                          errors.birthday?.month?.message
                        }
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
