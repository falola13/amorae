"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { Alert, Button, Field, Para, Sheet } from "@/components/ui/kit";
import { ShowButton } from "@/components/ui/onboarding-bits";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { changeEmailSchema, type ChangeEmailInput } from "@/lib/api/schemas";
import { notify } from "@/lib/store/toast";
import { useChangeEmail } from "../hooks";

/**
 * Changing the email changes what you log in with, so it asks for the
 * current password: a session left open on someone else's phone must not be
 * enough to take the account over. Field errors from the API (wrong
 * password, email taken) land on their inputs; anything else (e.g. too many
 * attempts) shows at the top.
 */
export function ChangeEmailSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const change = useChangeEmail();
  const [show, setShow] = useState(false);
  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors },
  } = useForm<ChangeEmailInput>({
    resolver: zodResolver(changeEmailSchema),
    defaultValues: { email: "", current_password: "" },
  });

  const close = () => {
    reset();
    setShow(false);
    onClose();
  };

  const onSubmit = (input: ChangeEmailInput) =>
    change.mutate(input, {
      onSuccess: () => {
        notify("Email updated. Use it the next time you log in.");
        close();
      },
      onError: (e) => {
        const fields = isApiError(e) ? e.fields : undefined;
        if (fields)
          for (const [k, m] of Object.entries(fields))
            setError(k as keyof ChangeEmailInput, { message: m });
        else setError("root", { message: isApiError(e) ? e.message : GENERIC_ERROR_MESSAGE });
      },
    });

  return (
    <Sheet open={open} onClose={close} title="Change your email" labelledBy="change-email-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <Para size="support">
          You&rsquo;ll log in with the new email from now on. Enter your password so we know
          it&rsquo;s you.
        </Para>
        {errors.root?.message ? <Alert message={errors.root.message} /> : null}
        <Field
          label="New email"
          type="email"
          autoComplete="email"
          inputMode="email"
          error={errors.email?.message}
          {...register("email")}
        />
        <Field
          label="Current password"
          type={show ? "text" : "password"}
          autoComplete="current-password"
          error={errors.current_password?.message}
          trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />}
          {...register("current_password")}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" loading={change.isPending}>
            Change email
          </Button>
          <Button type="button" variant="text" onClick={close}>
            Not now
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
