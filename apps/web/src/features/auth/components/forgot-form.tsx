"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Alert, Button, Field, LinkButton, Para } from "@/components/ui/kit";
import { Bottom } from "@/components/ui/onboarding-bits";
import { useForgotPassword } from "@/features/couple/hooks";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import { forgotSchema, type ForgotInput } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";

/**
 * The API answers the same whether or not the email has an account, so this
 * screen must too: "check your inbox" either way, never "no such account".
 */
export function ForgotForm() {
  const forgot = useForgotPassword();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ForgotInput>({ resolver: zodResolver(forgotSchema) });

  if (forgot.isSuccess)
    return (
      <div className="flex grow flex-col gap-4 px-6">
        <div role="status">
          <Para>
            If there&rsquo;s an account for that email, a link is on its way. It works for one hour.
            Check your spam folder if it doesn&rsquo;t arrive.
          </Para>
        </div>
        <Bottom>
          <LinkButton href={routes.login()}>Back to log in</LinkButton>
        </Bottom>
      </div>
    );

  const failure = forgot.error
    ? isApiError(forgot.error)
      ? forgot.error.message
      : GENERIC_ERROR_MESSAGE
    : null;

  return (
    <form
      method="post"
      onSubmit={handleSubmit((v) => forgot.mutate(v.email))}
      className="flex grow flex-col"
      noValidate
    >
      <div className="flex flex-col gap-[18px] px-6">
        {failure ? <Alert message={failure} /> : null}
        <Field
          label="Email"
          type="email"
          autoComplete="email"
          inputMode="email"
          error={errors.email?.message}
          {...register("email")}
        />
      </div>
      <Bottom>
        <Button type="submit" loading={forgot.isPending}>
          Send reset link
        </Button>
      </Bottom>
    </form>
  );
}
