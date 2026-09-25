import { isApiError } from "@/lib/api/errors";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";

// What a Server Action returns when it can't redirect: a form message plus
// per-field errors for setError(). Never echo a password back.
export type ActionError = { message: string; fields?: Record<string, string> };

export function toActionError(error: unknown): ActionError {
  if (isApiError(error)) return { message: error.message, fields: error.fields };
  return { message: GENERIC_ERROR_MESSAGE };
}
