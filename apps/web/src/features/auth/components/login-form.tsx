"use client";

import Link from "next/link";
import { useActionState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { idleFormState, type FormState } from "@/lib/forms";
import { loginAction } from "../actions";

interface LoginFormProps {
  next: string;
}

export function LoginForm({ next }: LoginFormProps) {
  const [state, formAction, pending] = useActionState<FormState, FormData>(
    loginAction,
    idleFormState,
  );

  return (
    <form action={formAction} className="flex flex-col gap-4" noValidate>
      <input type="hidden" name="next" value={next} />
      {state.status === "error" && state.message ? (
        <Alert message={state.message} />
      ) : null}
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
        autoComplete="current-password"
        minLength={8}
        maxLength={72}
        required
        error={state.fields?.password}
      />
      <Button type="submit" disabled={pending}>
        {pending ? "Signing in…" : "Sign in"}
      </Button>
      <p className="text-center text-sm text-fg-muted">
        Need an account?{" "}
        <Link
          href="/register"
          className="font-medium text-accent hover:underline"
        >
          Register
        </Link>
      </p>
    </form>
  );
}
