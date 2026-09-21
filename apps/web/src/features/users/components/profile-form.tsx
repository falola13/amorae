"use client";

import { useActionState } from "react";

import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { idleFormState, type FormState } from "@/lib/forms";
import { updateProfileAction } from "../actions";

interface ProfileFormProps {
  displayName: string;
}

export function ProfileForm({ displayName }: ProfileFormProps) {
  const [state, formAction, pending] = useActionState<FormState, FormData>(
    updateProfileAction,
    idleFormState,
  );

  return (
    <form action={formAction} className="flex flex-col gap-4" noValidate>
      {state.status === "error" && state.message ? <Alert message={state.message} /> : null}
      {state.status === "success" && state.message ? (
        <Alert variant="success" message={state.message} />
      ) : null}
      <Input
        label="Display name"
        name="display_name"
        type="text"
        autoComplete="name"
        maxLength={50}
        required
        defaultValue={state.values?.display_name ?? displayName}
        error={state.fields?.display_name}
      />
      <Button type="submit" disabled={pending}>
        {pending ? "Saving…" : "Save changes"}
      </Button>
    </form>
  );
}
