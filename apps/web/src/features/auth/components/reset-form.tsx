"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { Alert, Button, Field, LinkButton, Para } from "@/components/ui/kit";
import { Bottom, ShowButton } from "@/components/ui/onboarding-bits";
import { useResetPassword } from "@/features/couple/hooks";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { resetSchema, type ResetInput } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";

export function ResetForm({ token }: { token: string }) {
  const reset = useResetPassword();
  const [show, setShow] = useState(false);
  const [linkProblem, setLinkProblem] = useState<string | null>(
    token ? null : "This link is missing its code. Ask for a new one.",
  );
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<ResetInput>({ resolver: zodResolver(resetSchema) });

  if (reset.isSuccess)
    return (
      <div className="flex grow flex-col gap-4 px-6">
        <div role="status">
          <Para>Your password is changed. Log in with the new one.</Para>
        </div>
        <Bottom>
          <LinkButton href={routes.login()}>Log in</LinkButton>
        </Bottom>
      </div>
    );

  // A dead link can't be fixed on this screen, so it gets a way out, not a retry.
  if (linkProblem)
    return (
      <div className="flex grow flex-col gap-4 px-6">
        <Alert message={linkProblem} />
        <Bottom>
          <LinkButton href={routes.forgot}>Send a new link</LinkButton>
        </Bottom>
      </div>
    );

  const onSubmit = (v: ResetInput) =>
    reset.mutate(
      { token, new_password: v.new_password },
      {
        onError: (e) => {
          const fields = isApiError(e) ? e.fields : undefined;
          if (fields?.token) setLinkProblem(fields.token);
          else if (fields?.new_password) setError("new_password", { message: fields.new_password });
          else setError("root", { message: isApiError(e) ? e.message : GENERIC_ERROR_MESSAGE });
        },
      },
    );

  return (
    <form method="post" onSubmit={handleSubmit(onSubmit)} className="flex grow flex-col" noValidate>
      <div className="flex flex-col gap-[18px] px-6">
        {errors.root?.message ? <Alert message={errors.root.message} /> : null}
        <Field
          label="New password"
          type={show ? "text" : "password"}
          autoComplete="new-password"
          placeholder="At least 10 characters"
          error={errors.new_password?.message}
          trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />}
          {...register("new_password")}
        />
      </div>
      <Bottom>
        <Button type="submit" loading={reset.isPending}>
          Save new password
        </Button>
      </Bottom>
    </form>
  );
}
