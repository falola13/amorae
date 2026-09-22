import { isApiError } from "@/lib/api/errors";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";

// What a Server Action returns to a React Hook Form submit handler when it
// can't redirect: a message for the top of the form and per-field errors the
// caller maps onto setError(). Never echo a password back.
export type ActionError = { message: string; fields?: Record<string, string> };

export function toActionError(error: unknown): ActionError {
  if (isApiError(error)) return { message: error.message, fields: error.fields };
  return { message: GENERIC_ERROR_MESSAGE };
}
