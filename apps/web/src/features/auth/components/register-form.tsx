"use client";

import Link from "next/link";
import { useActionState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { idleFormState, type FormState } from "@/lib/forms";
import { registerAction } from "../actions";

export function RegisterForm() {
  const [state, formAction, pending] = useActionState<FormState, FormData>(registerAction, idleFormState);

  return (
    <form action={formAction} className="flex flex-col gap-4" noValidate>
      {state.status === "error" && state.message ? <Alert message={state.message} /> : null}
      <Input
        label="Display name"
        name="display_name"
        type="text"
        defaultValue={state.values?.display_name}
        autoComplete="name"
        maxLength={50}
        required
        error={state.fields?.display_name}
      />
      <Input
        label="Email"
        name="email"
        type="email"
        defaultValue={state.values?.email}
        autoComplete="email"
        required
        error={state.fields?.email}
      />
      <Input
        label="Password"
        name="password"
        type="password"
        autoComplete="new-password"
        minLength={8}
        maxLength={72}
        required
        error={state.fields?.password}
      />
      <Button type="submit" disabled={pending}>
        {pending ? "Creating account…" : "Create account"}
      </Button>
      <p className="text-center text-sm text-fg-muted">
        Already have an account?{" "}
        <Link href="/login" className="font-medium text-accent hover:underline">
          Sign in
        </Link>
      </p>
    </form>
  );
}
