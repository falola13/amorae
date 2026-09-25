"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useState, type ReactNode } from "react";
import { Controller, useForm, type Control } from "react-hook-form";

import { Icon } from "@/components/icons";
import { Alert, Button, Checkbox, Field, LinkButton } from "@/components/ui/kit";
import { Bottom, ShowButton } from "@/components/ui/onboarding-bits";
import { registerSchema, type RegisterInput } from "@/lib/api/schemas";
import { routes } from "@/lib/routes";
import { registerAction } from "../actions";

export function RegisterForm() {
  const [show, setShow] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    setError,
    control,
    formState: { errors, isSubmitting },
  } = useForm<RegisterInput>({
    resolver: zodResolver(registerSchema),
    defaultValues: { age_confirmed: false, accepted_terms: false, faith_consent: false },
  });

  const onSubmit = async (input: RegisterInput) => {
    setMessage(null);
    const err = await registerAction(input);
    if (err) {
      setMessage(err.message);
      for (const [k, v] of Object.entries(err.fields ?? {}))
        setError(k as keyof RegisterInput, { message: v });
    }
  };

  return (
    <form method="post" onSubmit={handleSubmit(onSubmit)} className="flex grow flex-col" noValidate>
      <div className="flex grow flex-col gap-[18px] px-6">
        {message ? <Alert message={message} /> : null}
        <Field
          label="Your first name"
          autoComplete="given-name"
          hint="This is what your partner will see."
          error={errors.display_name?.message}
          {...register("display_name")}
        />
        <Field
          label="Email"
          type="email"
          autoComplete="email"
          inputMode="email"
          error={errors.email?.message}
          {...register("email")}
        />
        <Field
          label="Password"
          type={show ? "text" : "password"}
          autoComplete="new-password"
          placeholder="At least 10 characters"
          error={errors.password?.message}
          trailing={<ShowButton shown={show} onClick={() => setShow((s) => !s)} />}
          {...register("password")}
        />
        <div className="flex flex-col">
          <ConsentBox control={control} name="age_confirmed" error={errors.age_confirmed?.message}>
            I&rsquo;m 18 or older
          </ConsentBox>
          <ConsentBox
            control={control}
            name="accepted_terms"
            error={errors.accepted_terms?.message}
          >
            {/* New tab, so reading them doesn't throw away what's been typed here. */}
            <span>
              I agree to the{" "}
              <Link
                href={routes.terms}
                target="_blank"
                rel="noopener noreferrer"
                className="font-semibold text-plum"
                onClick={(e) => e.stopPropagation()}
              >
                Terms
              </Link>{" "}
              and{" "}
              <Link
                href={routes.privacy}
                target="_blank"
                rel="noopener noreferrer"
                className="font-semibold text-plum"
                onClick={(e) => e.stopPropagation()}
              >
                Privacy Policy
              </Link>
            </span>
          </ConsentBox>
          <ConsentBox control={control} name="faith_consent">
            Show me faith content (prayers and scripture). Optional.
          </ConsentBox>
        </div>
      </div>
      <Bottom>
        <Button type="submit" loading={isSubmitting}>
          Continue
        </Button>
        <LinkButton href={routes.login()} variant="text">
          I already have an account
        </LinkButton>
      </Bottom>
    </form>
  );
}

function ConsentBox({
  control,
  name,
  error,
  children,
}: {
  control: Control<RegisterInput>;
  name: "age_confirmed" | "accepted_terms" | "faith_consent";
  error?: string;
  children: ReactNode;
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => (
        <div className="flex flex-col">
          <Checkbox checked={field.value} onChange={field.onChange}>
            {children}
          </Checkbox>
          {error ? (
            <div role="alert" className="flex items-center gap-1.5 pb-1 text-[13px] text-red">
              <Icon name="alert" size={16} />
              {error}
            </div>
          ) : null}
        </div>
      )}
    />
  );
}
