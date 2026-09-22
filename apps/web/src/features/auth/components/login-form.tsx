"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { Alert, Button, Field, LinkButton } from "@/components/ui/kit";
import { Bottom, ShowButton } from "@/components/ui/onboarding-bits";
import { loginSchema, type LoginInput } from "@/lib/api/schemas";
import { loginAction } from "../actions";
import { routes } from "@/lib/routes";

export function LoginForm({ next, expired }: { next: string; expired?: boolean }) {
  const [show, setShow] = useState(false);
  const [message, setMessage] = useState<string | null>(expired ? "Your session expired. Please log in again." : null);
  const { register, handleSubmit, setError, formState: { errors, isSubmitting } } = useForm<LoginInput>({ resolver: zodResolver(loginSchema), defaultValues: { next } });

  const onSubmit = async (input: LoginInput) => {
    setMessage(null);
    const err = await loginAction(input); // redirects on success
    if (err) {
      setMessage(err.message);
      for (const [k, v] of Object.entries(err.fields ?? {})) setError(k as keyof LoginInput, { message: v });
    }
  };

  return (
    <form method="post" onSubmit={handleSubmit(onSubmit)} className="flex grow flex-col" noValidate>
      <input type="hidden" {...register("next")} />
      <div className="flex grow flex-col gap-[18px] px-6">
        {message ? <Alert message={message} /> : null}
        <Field label="Email" type="email" autoComplete="email" inputMode="email" error={errors.email?.message} {...register("email")} />
        <Field label="Password" type={show ? "text" : "password"} autoComplete="current-password" error={errors.password?.message} trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />} {...register("password")} />
        {/* No reset flow exists yet (the API has no endpoint), so this is a plain
            note rather than a link to nowhere. Make it a link when it lands. */}
        <p className="m-0 -mt-2.5 flex h-11 items-center text-[15px] text-stone">Forgot your password? Reset is coming soon.</p>
      </div>
      <Bottom>
        <Button type="submit" loading={isSubmitting}>Log in</Button>
        <LinkButton href={routes.register} variant="text">Create an account instead</LinkButton>
      </Bottom>
    </form>
  );
}
