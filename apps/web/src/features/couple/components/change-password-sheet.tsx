"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { Alert, Button, Field, Para, Sheet } from "@/components/ui/kit";
import { ShowButton } from "@/components/ui/onboarding-bits";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { changePasswordSchema, type ChangePasswordInput } from "@/lib/api/schemas";
import { notify } from "@/lib/store/toast";
import { useChangePassword } from "../hooks";

// API signs out every other device on password change; this device stays signed in.
export function ChangePasswordSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const change = useChangePassword();
  const [show, setShow] = useState(false);
  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors, isDirty },
  } = useForm<ChangePasswordInput>({
    resolver: zodResolver(changePasswordSchema),
    defaultValues: { current_password: "", new_password: "" },
  });

  const close = () => {
    reset();
    setShow(false);
    onClose();
  };

  const onSubmit = (input: ChangePasswordInput) =>
    change.mutate(input, {
      onSuccess: () => {
        notify("Password changed. Your other devices are signed out.");
        close();
      },
      onError: (e) => {
        const fields = isApiError(e) ? e.fields : undefined;
        if (fields)
          for (const [k, m] of Object.entries(fields))
            setError(k as keyof ChangePasswordInput, { message: m });
        else setError("root", { message: isApiError(e) ? e.message : GENERIC_ERROR_MESSAGE });
      },
    });

  return (
    <Sheet
      open={open}
      onClose={close}
      dirty={isDirty}
      title="Change your password"
      labelledBy="change-password-h"
    >
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <Para size="support">Your other devices will be signed out. This one stays signed in.</Para>
        {errors.root?.message ? <Alert message={errors.root.message} /> : null}
        <Field
          label="Current password"
          type={show ? "text" : "password"}
          autoComplete="current-password"
          error={errors.current_password?.message}
          trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />}
          {...register("current_password")}
        />
        <Field
          label="New password"
          type={show ? "text" : "password"}
          autoComplete="new-password"
          placeholder="At least 10 characters"
          error={errors.new_password?.message}
          {...register("new_password")}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" loading={change.isPending}>
            Change password
          </Button>
          <Button type="button" variant="text" onClick={close}>
            Not now
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
