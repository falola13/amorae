"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Alert, Button, Field, Para, Sheet } from "@/components/ui/kit";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { isApiError } from "@/lib/api/errors";
import {
  DELETE_CONFIRMATION,
  deleteAccountSchema,
  isDeleteConfirmation,
  type DeleteAccountInput,
} from "@/lib/api/schemas";
import { useDeleteAccount } from "../hooks";

/**
 * Deleting an account can't be undone, so the person types the word
 * themselves; a stray tap on the button is never enough. The button stays
 * disabled until the word matches, and the API checks the same word again.
 */
export function DeleteAccountSheet({
  open,
  onClose,
  partner,
  onDeleted,
}: {
  open: boolean;
  onClose: () => void;
  partner?: string;
  onDeleted: () => Promise<void> | void;
}) {
  const del = useDeleteAccount();
  const {
    register,
    handleSubmit,
    setError,
    reset,
    watch,
    formState: { errors },
  } = useForm<DeleteAccountInput>({
    resolver: zodResolver(deleteAccountSchema),
    defaultValues: { confirm: "", current_password: "" },
  });
  const typed = isDeleteConfirmation(watch("confirm"));

  const close = () => {
    reset();
    onClose();
  };

  const onSubmit = (input: DeleteAccountInput) =>
    del.mutate(input, {
      onSuccess: onDeleted,
      onError: (e) => {
        const fields = isApiError(e) ? e.fields : undefined;
        if (fields?.confirm || fields?.current_password) {
          if (fields.confirm) setError("confirm", { message: fields.confirm });
          if (fields.current_password)
            setError("current_password", { message: fields.current_password });
        } else setError("root", { message: isApiError(e) ? e.message : GENERIC_ERROR_MESSAGE });
      },
    });

  return (
    <Sheet open={open} onClose={close} title="Delete your account?" labelledBy="del-h">
      <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
        <Para>
          This removes you from your space with {partner ?? "your partner"} and deletes everything
          you wrote. It can&rsquo;t be undone.
        </Para>
        {errors.root?.message ? <Alert message={errors.root.message} /> : null}
        <Field
          label={`Type “${DELETE_CONFIRMATION}” to confirm`}
          autoComplete="off"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          error={errors.confirm?.message}
          {...register("confirm")}
        />
        <Field
          label="Current password"
          type="password"
          autoComplete="current-password"
          error={errors.current_password?.message}
          {...register("current_password")}
        />
        <div className="flex flex-col gap-1">
          <Button
            type="submit"
            variant="secondary"
            className="border-red text-red"
            disabled={!typed}
            loading={del.isPending}
          >
            Delete my account
          </Button>
          <Button type="button" variant="text" onClick={close}>
            Keep my account
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
