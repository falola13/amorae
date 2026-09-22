"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { Alert, Button, Field } from "@/components/ui/kit";
import { Bottom, ShowButton } from "@/components/ui/onboarding-bits";
import { registerSchema, type RegisterInput } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";
import { registerAction } from "../actions";

export function RegisterForm() {
  const [show, setShow] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const { register, handleSubmit, setError, formState: { errors, isSubmitting } } = useForm<RegisterInput>({ resolver: zodResolver(registerSchema) });

  const onSubmit = async (input: RegisterInput) => {
    setMessage(null);
    const err = await registerAction(input);
    if (err) {
      setMessage(err.message);
      for (const [k, v] of Object.entries(err.fields ?? {})) setError(k as keyof RegisterInput, { message: v });
    }
  };

  return (
    <form method="post" onSubmit={handleSubmit(onSubmit)} className="flex grow flex-col" noValidate>
      <div className="flex grow flex-col gap-[18px] px-6">
        {message ? <Alert message={message} /> : null}
        <Field label="Your first name" autoComplete="given-name" hint="This is what your partner will see." error={errors.display_name?.message} {...register("display_name")} />
        <Field label="Email" type="email" autoComplete="email" inputMode="email" error={errors.email?.message} {...register("email")} />
        <Field label="Password" type={show ? "text" : "password"} autoComplete="new-password" placeholder="At least 10 characters" error={errors.password?.message} trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />} {...register("password")} />
      </div>
      <Bottom>
        <Button type="submit" loading={isSubmitting}>Continue</Button>
        {/* New tab, so reading them doesn't throw away what's been typed here. */}
        <div className="pt-1.5 text-center text-[13px] leading-normal text-stone">
          By continuing you agree to the{" "}
          <Link href={routes.terms} target="_blank" rel="noopener noreferrer" className="font-semibold text-plum">Terms</Link> and{" "}
          <Link href={routes.privacy} target="_blank" rel="noopener noreferrer" className="font-semibold text-plum">Privacy Policy</Link>.
        </div>
      </Bottom>
    </form>
  );
}
